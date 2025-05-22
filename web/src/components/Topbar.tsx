import React from 'react';
import { BellIcon } from 'lucide-react';
import ThemeButton from './ThemeButton';

export default function Topbar() {
  return (
    <header className="flex items-center justify-between bg-primary px-8 py-4 shadow">
      <h1 className="text-2xl font-bold text-primary-content">小智AI管理系统</h1>
      <div className="flex items-center space-x-4">
        <ThemeButton />
        <div className="avatar online">
          <div className="w-10 h-10 rounded-full ring ring-primary ring-offset-2 ring-offset-base-100">
            <img src="/assets/icons/icon.png" alt="User avatar" />
          </div>
        </div>
      </div>
    </header>
  );
}