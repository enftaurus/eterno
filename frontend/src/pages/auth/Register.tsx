import React, { useState, useEffect } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import { Card } from '../../components/ui/Card';
import { Button } from '../../components/ui/Button';
import { Input } from '../../components/ui/Input';
import { api } from '../../lib/api';
import { Code2 } from 'lucide-react';

interface InstitutionOption {
  id: string;
  name: string;
  slug: string;
}

export const Register: React.FC = () => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [role, setRole] = useState('STUDENT');
  const [institutionSlug, setInstitutionSlug] = useState('');
  const [institutions, setInstitutions] = useState<InstitutionOption[]>([]);
  const [departmentName, setDepartmentName] = useState('Computer Science');
  const [batchName, setBatchName] = useState('Batch of 2026');
  const [studentID, setStudentID] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const { register } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    const fetchInstitutions = async () => {
      try {
        const list = await api.get<InstitutionOption[]>('/auth/institutions');
        if (list && list.length > 0) {
          setInstitutions(list);
          setInstitutionSlug(list[0].slug);
        }
      } catch (err) {
        // Fallback gracefully without throwing
      }
    };
    fetchInstitutions();
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!institutionSlug) {
      setError('Please select or enter your institution slug');
      return;
    }

    setIsLoading(true);
    setError(null);
    try {
      await register({
        email,
        password,
        display_name: displayName,
        institution_slug: institutionSlug,
        role,
        department_name: departmentName,
        batch_name: batchName,
        student_id_number: studentID,
      });
      navigate('/');
    } catch (err: any) {
      setError(err.message || 'Registration failed');
    } finally {
      setIsLoading(false);
    }
  };

  const selectedInstName = institutions.find((i) => i.slug === institutionSlug)?.name;

  return (
    <div className="min-h-screen bg-[#fafafa] flex flex-col justify-center py-12 sm:px-6 lg:px-8">
      <div className="sm:mx-auto sm:w-full sm:max-w-md text-center">
        <div className="inline-flex items-center justify-center w-12 h-12 rounded-xl bg-indigo-600 text-white shadow-md mb-4">
          <Code2 className="w-7 h-7" />
        </div>
        <h2 className="text-2xl font-bold tracking-tight text-neutral-900">Create your Eterno Account</h2>
        <p className="mt-1 text-xs text-neutral-500">
          {selectedInstName ? `Enrolling into ${selectedInstName}` : 'Select your institution to enroll'}
        </p>
      </div>

      <div className="mt-8 sm:mx-auto sm:w-full sm:max-w-md">
        <Card className="shadow-lg border-neutral-200/80 p-6 sm:p-8">
          <form onSubmit={handleSubmit} className="space-y-4">
            {error && (
              <div className="p-3 bg-rose-50 border border-rose-200 text-rose-700 text-xs rounded-md">
                {error}
              </div>
            )}

            <div>
              <label className="block text-xs font-semibold text-neutral-700 uppercase tracking-wider mb-1.5">
                Institution
              </label>
              {institutions.length > 0 ? (
                <select
                  value={institutionSlug}
                  onChange={(e) => setInstitutionSlug(e.target.value)}
                  className="w-full rounded-md border border-neutral-300 text-sm px-3 py-2 bg-white text-neutral-900 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                >
                  {institutions.map((inst) => (
                    <option key={inst.id} value={inst.slug}>
                      {inst.name}
                    </option>
                  ))}
                </select>
              ) : (
                <Input
                  placeholder="e.g. stanford-engineering"
                  value={institutionSlug}
                  onChange={(e) => setInstitutionSlug(e.target.value)}
                  required
                />
              )}
            </div>

            <Input
              label="Full Display Name"
              placeholder="e.g. Maya Lin"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              required
            />

            <Input
              label="University Email"
              type="email"
              placeholder="developer@college.edu"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />

            <Input
              label="Password"
              type="password"
              placeholder="••••••••••••"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />

            <div>
              <label className="block text-xs font-semibold text-neutral-700 uppercase tracking-wider mb-1.5">Account Role</label>
              <select
                value={role}
                onChange={(e) => setRole(e.target.value)}
                className="w-full rounded-md border border-neutral-300 text-sm px-3 py-2 bg-white text-neutral-900 focus:outline-none focus:ring-2 focus:ring-indigo-500"
              >
                <option value="STUDENT">Student / Developer</option>
                <option value="COLLEGE_ADMIN">College Administrator</option>
              </select>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <Input
                label="Department"
                value={departmentName}
                onChange={(e) => setDepartmentName(e.target.value)}
              />
              <Input
                label="Batch / Year"
                value={batchName}
                onChange={(e) => setBatchName(e.target.value)}
              />
            </div>

            <Button type="submit" size="md" className="w-full mt-2" isLoading={isLoading}>
              Complete Enrollment
            </Button>
          </form>

          <div className="mt-6 text-center text-xs text-neutral-500">
            Already have an account?{' '}
            <Link to="/login" className="font-semibold text-indigo-600 hover:text-indigo-800">
              Sign in
            </Link>
          </div>
        </Card>
      </div>
    </div>
  );
};
