import React, { createContext, useContext, useState, useEffect } from 'react';
import { User, Institution } from '../types';
import { api } from '../lib/api';

interface AuthContextType {
  user: User | null;
  institution: Institution | null;
  token: string | null;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (data: any) => Promise<void>;
  logout: () => void;
  refreshUser: () => Promise<void>;
  isCollegeAdmin: boolean;
  isStudent: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [institution, setInstitution] = useState<Institution | null>(null);
  const [token, setToken] = useState<string | null>(localStorage.getItem('eterno_token'));
  const [isLoading, setIsLoading] = useState<boolean>(true);

  const fetchCurrentUser = async () => {
    try {
      const data = await api.get<{ user: User; institution?: Institution }>('/me');
      setUser(data.user);
      if (data.institution) {
        setInstitution(data.institution);
      }
    } catch (err) {
      console.error('Failed to load user session', err);
      logout();
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    if (token) {
      fetchCurrentUser();
    } else {
      setIsLoading(false);
    }
  }, [token]);

  const login = async (email: string, password: string) => {
    const res = await api.post<{ token: string; user: User; institution?: Institution }>('/auth/login', { email, password });
    localStorage.setItem('eterno_token', res.token);
    setToken(res.token);
    setUser(res.user);
    if (res.institution) {
      setInstitution(res.institution);
    }
  };

  const register = async (formData: any) => {
    const res = await api.post<{ token: string; user: User; institution?: Institution }>('/auth/register', formData);
    localStorage.setItem('eterno_token', res.token);
    setToken(res.token);
    setUser(res.user);
    if (res.institution) {
      setInstitution(res.institution);
    }
  };

  const logout = () => {
    localStorage.removeItem('eterno_token');
    setToken(null);
    setUser(null);
    setInstitution(null);
  };

  const refreshUser = async () => {
    if (token) {
      await fetchCurrentUser();
    }
  };

  const isCollegeAdmin = user?.role === 'COLLEGE_ADMIN' || user?.role === 'PLATFORM_ADMIN';
  const isStudent = user?.role === 'STUDENT';

  return (
    <AuthContext.Provider
      value={{
        user,
        institution,
        token,
        isLoading,
        login,
        register,
        logout,
        refreshUser,
        isCollegeAdmin,
        isStudent,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
