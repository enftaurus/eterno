import React, { useState, useEffect } from 'react';
import { Card, CardHeader } from '../../components/ui/Card';
import { Badge } from '../../components/ui/Badge';
import { Button } from '../../components/ui/Button';
import { Input } from '../../components/ui/Input';
import { Modal } from '../../components/ui/Modal';
import { api } from '../../lib/api';
import { Users, Upload, Download, Search, CheckCircle2, AlertCircle, FileSpreadsheet } from 'lucide-react';

interface StudentItem {
  id: string;
  email: string;
  display_name: string;
  department_name?: string | null;
  batch_name?: string | null;
  student_id_number?: string | null;
  leetcode_username?: string | null;
  leetcode_verification?: string | null;
  last_sync_at?: string | null;
  verified_solves_count: number;
  total_points: number;
  completed_assigned_count: number;
  joined_at: string;
}

export const AdminStudents: React.FC = () => {
  const [students, setStudents] = useState<StudentItem[]>([]);
  const [total, setTotal] = useState(0);
  const [search, setSearch] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [showImportModal, setShowImportModal] = useState(false);
  const [importFile, setImportFile] = useState<File | null>(null);
  const [isImporting, setIsImporting] = useState(false);
  const [importResult, setImportResult] = useState<{ imported: number; skipped: number; errors: string[] } | null>(null);

  const fetchStudents = async () => {
    setIsLoading(true);
    try {
      const url = `/admin/students?search=${encodeURIComponent(search)}`;
      const data = await api.get<{ students: StudentItem[]; total: number }>(url);
      setStudents(data.students || []);
      setTotal(data.total || 0);
    } catch (err) {
      console.error('Failed to load students', err);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchStudents();
  }, [search]);

  const handleImportSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!importFile) return;

    setIsImporting(true);
    setImportResult(null);

    const formData = new FormData();
    formData.append('file', importFile);

    try {
      const res = await api.upload<any>('/admin/students/import', formData);
      setImportResult(res);
      await fetchStudents();
    } catch (err: any) {
      alert(err.message || 'Import failed');
    } finally {
      setIsImporting(false);
    }
  };

  const handleExportCSV = () => {
    if (students.length === 0) return;
    const headers = ['Email', 'Name', 'Department', 'Batch', 'Student ID', 'LeetCode Username', 'Verified Solves', 'Total Points'];
    const rows = students.map((s) => [
      s.email,
      s.display_name,
      s.department_name || '',
      s.batch_name || '',
      s.student_id_number || '',
      s.leetcode_username || '',
      s.verified_solves_count,
      s.total_points,
    ]);

    const csvContent = 'data:text/csv;charset=utf-8,' + [headers, ...rows].map((e) => e.join(',')).join('\n');
    const encodedUri = encodeURI(csvContent);
    const link = document.createElement('a');
    link.setAttribute('href', encodedUri);
    link.setAttribute('download', `eterno-students-${new Date().toISOString().slice(0, 10)}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-bold text-neutral-900 tracking-tight">Student Directory & Enrollment</h1>
          <p className="text-xs text-neutral-500 mt-0.5">
            Manage student records, LeetCode verification states, and cohort memberships ({total} enrolled).
          </p>
        </div>

        <div className="flex items-center gap-2.5">
          <Button variant="outline" size="sm" onClick={handleExportCSV} icon={<Download className="w-3.5 h-3.5" />}>
            Export CSV
          </Button>

          <Button size="sm" onClick={() => setShowImportModal(true)} icon={<Upload className="w-3.5 h-3.5" />}>
            Import Students CSV
          </Button>
        </div>
      </div>

      {/* Search and Filters */}
      <div className="flex items-center gap-3 max-w-md">
        <Input
          placeholder="Search by name, email, or student ID..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          icon={<Search className="w-4 h-4 text-neutral-400" />}
        />
      </div>

      {/* Students Table */}
      <Card>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead>
              <tr className="border-b border-neutral-200 text-neutral-400 uppercase tracking-wider text-[10px]">
                <th className="py-3 px-4 font-semibold">Student Name & Email</th>
                <th className="py-3 px-4 font-semibold">Department & Batch</th>
                <th className="py-3 px-4 font-semibold">Student ID</th>
                <th className="py-3 px-4 font-semibold">LeetCode Account</th>
                <th className="py-3 px-4 font-semibold text-right">Verified Solves</th>
                <th className="py-3 px-4 font-semibold text-right">Points</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-neutral-100 text-neutral-700">
              {isLoading ? (
                <tr>
                  <td colSpan={6} className="text-center py-12 text-neutral-400">Loading student directory...</td>
                </tr>
              ) : students.length === 0 ? (
                <tr>
                  <td colSpan={6} className="text-center py-12 text-neutral-400">No students found matching query.</td>
                </tr>
              ) : (
                students.map((s) => (
                  <tr key={s.id} className="hover:bg-neutral-50/50 transition-colors">
                    <td className="py-3 px-4">
                      <div className="font-semibold text-neutral-900">{s.display_name}</div>
                      <div className="text-[11px] text-neutral-400">{s.email}</div>
                    </td>
                    <td className="py-3 px-4 text-neutral-500">
                      <div>{s.department_name || 'General CS'}</div>
                      <div className="text-[10px] text-neutral-400">{s.batch_name || 'Class of 2026'}</div>
                    </td>
                    <td className="py-3 px-4 font-mono text-[11px] text-neutral-600">
                      {s.student_id_number || '—'}
                    </td>
                    <td className="py-3 px-4">
                      {s.leetcode_username ? (
                        <div className="flex items-center gap-1.5">
                          <span className="font-mono text-[11px] text-neutral-800">@{s.leetcode_username}</span>
                          <Badge variant={s.leetcode_verification === 'VERIFIED' ? 'success' : 'warning'} size="sm" dot>
                            {s.leetcode_verification}
                          </Badge>
                        </div>
                      ) : (
                        <span className="text-[11px] text-neutral-400 italic">Not connected</span>
                      )}
                    </td>
                    <td className="py-3 px-4 text-right font-medium text-neutral-800">
                      {s.verified_solves_count} solved
                    </td>
                    <td className="py-3 px-4 text-right font-bold text-indigo-600">
                      {s.total_points} pts
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </Card>

      {/* Modal: CSV Import */}
      <Modal
        isOpen={showImportModal}
        onClose={() => {
          setShowImportModal(false);
          setImportFile(null);
          setImportResult(null);
        }}
        title="Import Students from CSV"
        description="Upload a CSV spreadsheet to bulk enroll students, assign departments and batches, and associate LeetCode usernames."
      >
        <form onSubmit={handleImportSubmit} className="space-y-4">
          <div className="p-4 bg-neutral-50 rounded-lg border border-neutral-200/80 text-xs text-neutral-600 space-y-1">
            <p className="font-semibold text-neutral-800">Expected CSV Header Format:</p>
            <code className="block p-2 bg-white rounded border border-neutral-200 font-mono text-[11px] text-neutral-900">
              email, display_name, department, batch, student_id, leetcode_username
            </code>
          </div>

          <div>
            <input
              type="file"
              accept=".csv"
              onChange={(e) => setImportFile(e.target.files?.[0] || null)}
              className="block w-full text-xs text-neutral-500 file:mr-4 file:py-2 file:px-4 file:rounded-md file:border-0 file:text-xs file:font-semibold file:bg-indigo-50 file:text-indigo-700 hover:file:bg-indigo-100"
              required
            />
          </div>

          {importResult && (
            <div className="p-3 bg-neutral-50 border border-neutral-200 rounded-md text-xs space-y-1.5">
              <p className="font-semibold text-neutral-900">
                Imported: {importResult.imported} | Skipped/Failed: {importResult.skipped}
              </p>
              {importResult.errors && importResult.errors.length > 0 && (
                <div className="text-rose-600 space-y-0.5 text-[11px] max-h-32 overflow-y-auto">
                  {importResult.errors.map((err, i) => (
                    <div key={i}>{err}</div>
                  ))}
                </div>
              )}
            </div>
          )}

          <div className="flex items-center justify-end gap-2.5 pt-3 border-t border-neutral-100">
            <Button variant="ghost" size="sm" type="button" onClick={() => setShowImportModal(false)}>
              Close
            </Button>
            <Button size="sm" type="submit" isLoading={isImporting} disabled={!importFile}>
              Upload & Process CSV
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
};
