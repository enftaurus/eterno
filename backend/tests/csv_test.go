package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/eterno/backend/internal/institutions"
)

// Tests 61-75: CSV Import/Export Hardening, Formula Injection, RFC 4180, and Bounds Enforcement

func TestCSVImportValid(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	instID, _ := createTestUserAndInst(t, db)
	svc := institutions.NewService(db)

	uid1 := fmt.Sprintf("%d", time.Now().UnixNano())
	uid2 := fmt.Sprintf("%d", time.Now().UnixNano()+1)
	csvData := fmt.Sprintf(`email,display_name,department,batch,student_id,leetcode_username
alice_%s@college.edu,Alice Smith,Computer Science,Batch 2026,CS-101,alicesmith_%s
bob_%s@college.edu,Bob Jones,Computer Science,Batch 2026,CS-102,bobjones_%s
`, uid1, uid1, uid2, uid2)

	res, err := svc.ImportStudentsCSV(ctx, instID, strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("ImportStudentsCSV failed: %v", err)
	}

	if res.TotalRows != 2 || res.Imported != 2 || res.Skipped != 0 {
		t.Errorf("unexpected import result: total=%d, imported=%d, skipped=%d, errors=%v",
			res.TotalRows, res.Imported, res.Skipped, res.Errors)
	}

	// Verify database record
	var name string
	err = db.QueryRowContext(ctx, "SELECT display_name FROM users WHERE email = $1", fmt.Sprintf("alice_%s@college.edu", uid1)).Scan(&name)
	if err != nil || name != "Alice Smith" {
		t.Errorf("user not created correctly: err=%v, name=%s", err, name)
	}

	// Verify unverified platform account
	var lcUser, lcStatus string
	err = db.QueryRowContext(ctx, `
		SELECT pa.username, pa.verification_status
		FROM platform_accounts pa
		JOIN users u ON pa.user_id = u.id
		WHERE u.email = $1
	`, fmt.Sprintf("alice_%s@college.edu", uid1)).Scan(&lcUser, &lcStatus)
	if err != nil || lcStatus != "UNVERIFIED" {
		t.Errorf("platform account incorrect: err=%v, lcStatus=%s", err, lcStatus)
	}
}

func TestCSVFormulaInjectionNeutralization(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	instID, _ := createTestUserAndInst(t, db)
	svc := institutions.NewService(db)

	uid := fmt.Sprintf("%d", time.Now().UnixNano())
	formulaPayloads := []struct {
		inName   string
		expected string
		email    string
	}{
		{inName: "=SUM(A1:A10)", expected: "'=SUM(A1:A10)", email: fmt.Sprintf("formula_eq_%s@college.edu", uid)},
		{inName: "+cmd|' /C calc'!A0", expected: "'+cmd|' /C calc'!A0", email: fmt.Sprintf("formula_plus_%s@college.edu", uid)},
		{inName: "-5+2", expected: "'-5+2", email: fmt.Sprintf("formula_minus_%s@college.edu", uid)},
		{inName: "@hyperlink('http://evil.com')", expected: "'@hyperlink('http://evil.com')", email: fmt.Sprintf("formula_at_%s@college.edu", uid)},
	}

	var sb strings.Builder
	sb.WriteString("email,display_name\n")
	for _, fp := range formulaPayloads {
		sb.WriteString(fmt.Sprintf("%s,\"%s\"\n", fp.email, fp.inName))
	}

	res, err := svc.ImportStudentsCSV(ctx, instID, strings.NewReader(sb.String()))
	if err != nil {
		t.Fatalf("ImportStudentsCSV failed: %v", err)
	}
	if res.Imported != len(formulaPayloads) {
		t.Errorf("expected %d imported, got %d (errors: %v)", len(formulaPayloads), res.Imported, res.Errors)
	}

	// Verify each was sanitized with leading single quote to neutralize spreadsheet execution
	for _, fp := range formulaPayloads {
		var storedName string
		err := db.QueryRowContext(ctx, "SELECT display_name FROM users WHERE email = $1", fp.email).Scan(&storedName)
		if err != nil {
			t.Errorf("failed to query %s: %v", fp.email, err)
			continue
		}
		if storedName != fp.expected {
			t.Errorf("Formula injection not sanitized for %s! Stored=%q, Expected=%q", fp.inName, storedName, fp.expected)
		}
	}
}

func TestCSVInvalidEmailRejection(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	instID, _ := createTestUserAndInst(t, db)
	svc := institutions.NewService(db)

	uid := fmt.Sprintf("%d", time.Now().UnixNano())
	csvData := fmt.Sprintf(`email,display_name
not-an-email,Bad Email Student
missing-domain@,No Domain Student
@missing-user.com,No User Student
valid_%s@college.edu,Good Student
`, uid)

	res, err := svc.ImportStudentsCSV(ctx, instID, strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("ImportStudentsCSV unexpected fatal error: %v", err)
	}

	if res.TotalRows != 4 {
		t.Errorf("expected 4 total rows, got %d", res.TotalRows)
	}
	if res.Imported != 1 {
		t.Errorf("expected 1 imported row, got %d", res.Imported)
	}
	if res.Skipped != 3 {
		t.Errorf("expected 3 skipped rows, got %d", res.Skipped)
	}
	if len(res.Errors) != 3 {
		t.Errorf("expected 3 row errors, got %d: %v", len(res.Errors), res.Errors)
	}
}

func TestCSVDuplicateEmailWithinBatchRejection(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	instID, _ := createTestUserAndInst(t, db)
	svc := institutions.NewService(db)

	uid := fmt.Sprintf("%d", time.Now().UnixNano())
	dupEmail := fmt.Sprintf("dup_%s@college.edu", uid)

	csvData := fmt.Sprintf(`email,display_name
%s,First Occurrence
%s,Second Occurrence (Duplicate)
unique_%s@college.edu,Unique Student
`, dupEmail, dupEmail, uid)

	res, err := svc.ImportStudentsCSV(ctx, instID, strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}

	if res.TotalRows != 3 {
		t.Errorf("expected 3 total rows, got %d", res.TotalRows)
	}
	if res.Imported != 2 {
		t.Errorf("expected 2 imported rows, got %d", res.Imported)
	}
	if res.Skipped != 1 {
		t.Errorf("expected 1 skipped duplicate row, got %d", res.Skipped)
	}

	// Verify error mentions duplicate and row number
	foundDupErr := false
	for _, e := range res.Errors {
		if strings.Contains(e, "duplicate email") && strings.Contains(e, "row 2") {
			foundDupErr = true
			break
		}
	}
	if !foundDupErr {
		t.Errorf("expected error to mention duplicate email and row 2, got: %v", res.Errors)
	}
}

func TestCSVBatchSizeLimitEnforcement(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	instID, _ := createTestUserAndInst(t, db)
	svc := institutions.NewService(db)

	// Create CSV with 1002 rows (1 header + 1001 students, exceeding 1000 limit)
	var sb strings.Builder
	sb.WriteString("email,display_name\n")
	for i := 0; i < 1001; i++ {
		sb.WriteString(fmt.Sprintf("student%d@college.edu,Student %d\n", i, i))
	}

	_, err := svc.ImportStudentsCSV(ctx, instID, strings.NewReader(sb.String()))
	if err == nil {
		t.Fatalf("expected error for exceeding 1000 record batch limit, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds maximum batch limit") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCSVEmptyAndHeaderOnlyRejection(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	instID, _ := createTestUserAndInst(t, db)
	svc := institutions.NewService(db)

	// Header only
	res, err := svc.ImportStudentsCSV(ctx, instID, strings.NewReader("email,display_name\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Imported != 0 || len(res.Errors) == 0 {
		t.Errorf("expected 0 imported and error message, got: imported=%d, errs=%v", res.Imported, res.Errors)
	}
}

func TestCSVQuotesAndCommasHandling(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	instID, _ := createTestUserAndInst(t, db)
	svc := institutions.NewService(db)

	uid := fmt.Sprintf("%d", time.Now().UnixNano())
	csvData := fmt.Sprintf(`email,display_name,department,batch,student_id,leetcode_username
"quoted_%s@college.edu","Smith, Alice M.","Computer Science, AI Track","Batch 2026, Section A","CS,101","alice_quoted_%s"
`, uid, uid)

	res, err := svc.ImportStudentsCSV(ctx, instID, strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if res.Imported != 1 {
		t.Errorf("expected 1 imported row, got %d (errors: %v)", res.Imported, res.Errors)
	}

	var name string
	err = db.QueryRowContext(ctx, "SELECT display_name FROM users WHERE email = $1", fmt.Sprintf("quoted_%s@college.edu", uid)).Scan(&name)
	if err != nil || name != "Smith, Alice M." {
		t.Errorf("name not parsed properly: err=%v, name=%s", err, name)
	}
}
