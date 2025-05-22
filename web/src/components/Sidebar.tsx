import React from 'react';
import { Link, useLocation } from 'react-router-dom';

const menuItems = [
  { label: '仪表盘', path: '/' },
  { label: '设备列表', path: '/devices' },
  { label: '设置', path: '/settings' },
];

export default function Sidebar() {
  const { pathname } = useLocation();
  return (
    <aside className="w-64 bg-base-200 p-4 flex flex-col">
      <h2 className="text-2xl font-bold mb-6 text-primary">我的应用</h2>
      <nav className="flex-1">
        <ul className="menu menu-vertical">
          {menuItems.map(item => (
            <li key={item.path} className={pathname === item.path ? 'active' : ''}>
              <Link to={item.path}>{item.label}</Link>
            </li>
          ))}
        </ul>
      </nav>
      <div className="mt-auto">
        <button className="btn btn-outline btn-sm w-full">退出登录</button>
      </div>
    </aside>
  );
}