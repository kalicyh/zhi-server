import React from 'react';

export default function Dashboard() {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      {['用户', '设备', '访问量'].map((title, idx) => (
        <div key={idx} className="card bg-base-100 shadow-lg">
          <div className="card-body">
            <h2 className="card-title">{title}</h2>
            <p className="text-3xl font-bold">{Math.floor(Math.random() * 1000)}</p>
          </div>
        </div>
      ))}
    </div>
  );
}