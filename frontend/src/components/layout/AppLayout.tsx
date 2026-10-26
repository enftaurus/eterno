import React, { useState, useEffect } from 'react';
import { Outlet, useLocation } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { useAuth } from '../../context/AuthContext';
import { Bell, CheckCircle2, AlertCircle, Sparkles } from 'lucide-react';
import { api } from '../../lib/api';
import { Notification } from '../../types';

export const AppLayout: React.FC = () => {
  const { user, institution, isCollegeAdmin } = useAuth();
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [showNotifications, setShowNotifications] = useState<boolean>(false);
  const location = useLocation();

  const fetchNotifications = async () => {
    try {
      const data = await api.get<Notification[]>('/me/notifications');
      setNotifications(data || []);
    } catch (e) {
      // silently handle
    }
  };

  useEffect(() => {
    fetchNotifications();
    const timer = setInterval(fetchNotifications, 30000); // 30s poll
    return () => clearInterval(timer);
  }, []);

  const unreadCount = notifications.filter((n) => !n.is_read).length;

  return (
    <div className="flex min-h-screen bg-[#fafafa]">
      <Sidebar />

      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {/* Top Header */}
        <header className="h-14 bg-white border-b border-neutral-200/80 px-6 flex items-center justify-between shrink-0 sticky top-0 z-20">
          <div className="flex items-center gap-3">
            <span className="text-xs font-medium text-neutral-400 capitalize">
              {location.pathname.replace('/', '').replace(/-/g, ' ') || 'Overview'}
            </span>
          </div>

          <div className="flex items-center gap-4">
            {/* Notifications Dropdown */}
            <div className="relative">
              <button
                onClick={() => setShowNotifications(!showNotifications)}
                className="relative p-1.5 text-neutral-500 hover:text-neutral-800 hover:bg-neutral-100 rounded-md transition-colors"
                title="Notifications"
              >
                <Bell className="w-4 h-4" />
                {unreadCount > 0 && (
                  <span className="absolute top-1 right-1 w-2 h-2 bg-indigo-600 rounded-full ring-2 ring-white" />
                )}
              </button>

              {showNotifications && (
                <div className="absolute right-0 mt-2 w-80 bg-white rounded-lg border border-neutral-200 shadow-lg py-2 z-50">
                  <div className="px-4 py-2 border-b border-neutral-100 flex items-center justify-between">
                    <span className="text-xs font-semibold text-neutral-900">Notifications</span>
                    <span className="text-[10px] text-neutral-400">{notifications.length} recent</span>
                  </div>

                  <div className="max-h-64 overflow-y-auto">
                    {notifications.length === 0 ? (
                      <p className="px-4 py-6 text-center text-xs text-neutral-400">No new notifications</p>
                    ) : (
                      notifications.map((n) => (
                        <div key={n.id} className="px-4 py-2.5 hover:bg-neutral-50 transition-colors border-b border-neutral-50 last:border-none">
                          <p className="text-xs font-medium text-neutral-800">{n.title}</p>
                          <p className="text-[11px] text-neutral-500 mt-0.5 line-clamp-2">{n.message}</p>
                          <span className="text-[10px] text-neutral-400 mt-1 block">
                            {new Date(n.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                          </span>
                        </div>
                      ))
                    )}
                  </div>
                </div>
              )}
            </div>

            {/* Institution Badge */}
            <div className="flex items-center gap-2 pl-3 border-l border-neutral-200/80 text-xs">
              <span className="w-2 h-2 rounded-full bg-emerald-500" />
              <span className="font-medium text-neutral-700">{institution?.name || 'Stanford Engineering'}</span>
            </div>
          </div>
        </header>

        {/* Main Workspace Body */}
        <main className="flex-1 overflow-y-auto p-6 md:p-8 max-w-7xl w-full mx-auto">
          <Outlet />
        </main>
      </div>
    </div>
  );
};
