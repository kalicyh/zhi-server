import React from 'react';
import { Link, useLocation } from 'react-router-dom';

const BASE_PATH = '/admin';

const menuItems = [
  { label: '仪表盘', path: '' },
  { label: '设备列表', path: 'devices' },
  { label: '设置', path: 'settings' },
  { label: '色卡', path: 'color' },
];

export default function Sidebar() {
  const { pathname } = useLocation();

  return (
    <aside className="w-54 bg-primary p-4 flex flex-col shadow-lg">
      <nav className="flex-1">
        <ul className="menu menu-vertical space-y-2 w-46">
          {menuItems.map(({ label, path }) => {
            const fullPath = `${BASE_PATH}/${path}`;
            const isActive = pathname === fullPath;
            return (
              <li
                key={fullPath}
                className={`w-full rounded-lg ${
                  isActive
                    ? 'bg-secondary text-white'
                    : 'hover:bg-secondary-focus hover:text-white'
                } transition-colors`}
              >
                <Link to={fullPath}>{label}</Link>
              </li>
            );
          })}
        </ul>
      </nav>
      <div className="mt-auto">
        <button className="btn btn-outline btn-sm w-full rounded-lg">退出登录</button>
      </div>
    </aside>
  );
}
