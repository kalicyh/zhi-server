import React from 'react';
import { Link, useLocation } from 'react-router-dom';

const menuItems = [
  { label: '仪表盘', path: '/' },
  { label: '设备列表', path: '/devices' },
  { label: '设置', path: '/settings' },
  { label: '色卡', path: '/color' },
];

export default function Sidebar() {
  const { pathname } = useLocation();
  return (
    <aside className="w-54 bg-primary p-4 flex flex-col shadow-lg">
      <nav className="flex-1">
        <ul className="menu menu-vertical space-y-2 w-46">
          {menuItems.map(item => (
            <li key={item.path} className={`w-full rounded-lg ${pathname === item.path ? 'bg-secondary text-white' : 'hover:bg-secondary-focus hover:text-white'} transition-colors`}>
              <Link to={item.path}>{item.label}</Link>
            </li>
          ))}
        </ul>
      </nav>
      <div className="mt-auto">
        <button className="btn btn-outline btn-sm w-full rounded-lg">退出登录</button>
      </div>
    </aside>
  );
}