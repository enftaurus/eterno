import React from 'react';
import { NavLink } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import {
  LayoutDashboard,
  Code2,
  Trophy,
  Activity,
  User as UserIcon,
  Link as LinkIcon,
  CheckCircle2,
  Users,
  Settings,
  ShieldAlert,
  FileSpreadsheet,
  LogOut,
  Building,
} from 'lucide-react';

export const Sidebar: React.FC = () => {
  const { user, institution, logout, isCollegeAdmin } = useAuth();

  const studentNavItems = [
    { to: '/', label: 'Overview', icon: LayoutDashboard },
    { to: '/challenges', label: 'Challenges', icon: Code2 },
    { to: '/submissions', label: 'My Submissions', icon: CheckCircle2 },
    { to: '/leaderboards', label: 'Leaderboards', icon: Trophy },
    { to: '/activity', label: 'Activity Heatmap', icon: Activity },
    { to: '/profile', label: 'Developer Profile', icon: UserIcon },
    { to: '/settings/connected-accounts', label: 'Connected Accounts', icon: LinkIcon },
  ];

  const adminNavItems = [
    { to: '/admin', label: 'Overview Dashboard', icon: LayoutDashboard },
    { to: '/admin/challenges', label: 'Challenges', icon: Code2 },
    { to: '/admin/students', label: 'Students Directory', icon: Users },
    { to: '/admin/groups', label: 'Batches & Groups', icon: Building },
    { to: '/admin/leaderboards', label: 'College Leaderboards', icon: Trophy },
    { to: '/admin/consistency', label: 'Consistency Analytics', icon: Activity },
    { to: '/admin/audit-logs', label: 'Audit Logs', icon: ShieldAlert },
  ];

  const activeClasses = 'bg-indigo-50/80 text-indigo-700 font-medium';
  const inactiveClasses = 'text-neutral-600 hover:text-neutral-900 hover:bg-neutral-50/80 transition-colors';

  return (
    <aside className="w-64 min-h-screen bg-white border-r border-neutral-200/80 flex flex-col justify-between shrink-0 select-none">
      <div>
        {/* Brand Header */}
        <div className="p-5 border-b border-neutral-100 flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-indigo-600 flex items-center justify-center text-white shadow-xs">
              <Code2 className="w-5 h-5" />
            </div>
            <div>
              <span className="text-base font-bold text-neutral-900 tracking-tight flex items-center gap-1.5">
                Eterno
                <span className="text-[10px] font-semibold uppercase px-1.5 py-0.2 bg-neutral-100 text-neutral-600 rounded">v1</span>
              </span>
              <p className="text-[11px] text-neutral-400 font-normal truncate max-w-[140px]">
                {institution?.name || 'Developer Intelligence'}
              </p>
            </div>
          </div>
        </div>

        {/* Navigation Section */}
        <div className="p-3">
          <p className="px-3 py-1.5 text-[11px] font-semibold text-neutral-400 uppercase tracking-wider">
            {isCollegeAdmin ? 'Institution Portal' : 'Developer Platform'}
          </p>

          <nav className="space-y-0.5 mt-1">
            {(isCollegeAdmin ? adminNavItems : studentNavItems).map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.to === '/' || item.to === '/admin'}
                className={({ isActive }) =>
                  `flex items-center gap-2.5 px-3 py-2 text-xs rounded-md ${
                    isActive ? activeClasses : inactiveClasses
                  }`
                }
              >
                <item.icon className="w-4 h-4 shrink-0" />
                <span>{item.label}</span>
              </NavLink>
            ))}
          </nav>

          {isCollegeAdmin && (
            <div className="mt-6 pt-4 border-t border-neutral-100">
              <p className="px-3 py-1 text-[11px] font-semibold text-neutral-400 uppercase tracking-wider">
                Student View
              </p>
              <nav className="space-y-0.5 mt-1">
                <NavLink
                  to="/"
                  className={({ isActive }) =>
                    `flex items-center gap-2.5 px-3 py-2 text-xs rounded-md ${
                      isActive ? activeClasses : inactiveClasses
                    }`
                  }
                >
                  <LayoutDashboard className="w-4 h-4 shrink-0" />
                  <span>Student Experience</span>
                </NavLink>
              </nav>
            </div>
          )}
        </div>
      </div>

      {/* User Footer */}
      <div className="p-3 border-t border-neutral-100">
        <div className="flex items-center justify-between p-2 rounded-lg bg-neutral-50/70 border border-neutral-200/60">
          <div className="flex items-center gap-2.5 min-w-0">
            <div className="w-7 h-7 rounded-full bg-neutral-200 text-neutral-700 flex items-center justify-center text-xs font-semibold shrink-0">
              {user?.display_name?.charAt(0) || 'U'}
            </div>
            <div className="min-w-0">
              <p className="text-xs font-medium text-neutral-900 truncate">{user?.display_name}</p>
              <p className="text-[10px] text-neutral-400 truncate">{user?.email}</p>
            </div>
          </div>

          <button
            onClick={logout}
            title="Log Out"
            className="p-1.5 text-neutral-400 hover:text-neutral-700 hover:bg-neutral-200/50 rounded-md transition-colors"
          >
            <LogOut className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>
    </aside>
  );
};
