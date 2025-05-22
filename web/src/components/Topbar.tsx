import React from 'react';
import { BellIcon } from 'lucide-react';
import ThemeButton from './ThemeButton';

export default function Topbar() {
  return (
    <header className="flex items-center justify-between bg-base-200 px-6 py-4 shadow">
      <h1 className="text-xl font-semibold">仪表盘</h1>
      <div className="flex items-center space-x-4">
        <ThemeButton />
        <div className="avatar online">
          <div className="w-8 rounded-full">
            <img src="/assets/icons/icon.png" alt="User avatar" />
          </div>
        </div>
      </div>
    </header>
  );
}