import React, { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import botIcon from '../assets/icons/bot.png';

export default function Login() {
  const [showLoginForm, setShowLoginForm] = useState(false);

  return (
    <div className="relative min-h-screen overflow-hidden bg-gradient-to-r from-primary via-secondary to-accent">
      {/* 漂浮模糊光斑背景 */}
      <div className="absolute inset-0 z-0 pointer-events-none">
        <div className="absolute w-72 h-72 bg-pink-400 rounded-full opacity-20 blur-3xl animate-blob1"></div>
        <div className="absolute w-96 h-96 bg-blue-400 rounded-full opacity-20 blur-3xl animate-blob2"></div>
        <div className="absolute w-80 h-80 bg-purple-500 rounded-full opacity-20 blur-3xl animate-blob3"></div>
      </div>

      {/* 内容区 */}
      <div
          className="relative z-10 flex items-center justify-center min-h-screen p-6"
          onClick={() => {
            if (showLoginForm) setShowLoginForm(false);
          }}
        >
        <AnimatePresence mode="wait">
          {!showLoginForm ? (
            <motion.div
              key="intro"
              className="text-white text-center max-w-2xl"
              initial={{ opacity: 0, y: 40 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -40, transition: { duration: 0.5 } }}
              transition={{ duration: 0.6, ease: 'easeInOut' }}
            >
              <img src={botIcon} className="mx-auto w-48 mb-6" alt="Logo" />
              <h1 className="text-5xl font-bold mb-4">欢迎来到 小智管理后台！</h1>
              <p className="text-lg mb-8">
                支持在线升级，快速、安全、便捷，助你更高效管理智能终端。
              </p>
              <button
                className="btn btn-neutral px-10 py-3 text-lg rounded-full shadow-lg"
                onClick={() => setShowLoginForm(true)}
              >
                立即进入
              </button>
            </motion.div>
          ) : (
            <motion.div
              key="login"
              className="w-full max-w-md bg-white/10 backdrop-blur-xl p-8 rounded-2xl shadow-2xl border border-white/20 text-white"
              onClick={(e) => e.stopPropagation()}
              initial={{ opacity: 0, y: 40 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -40, transition: { duration: 0.5 } }}
              transition={{ duration: 0.6, ease: 'easeInOut' }}
            >
              <h2 className="text-3xl font-bold mb-6 text-center">登录</h2>
              <form className="space-y-5">
                <div>
                  <label className="block text-sm mb-1">用户名</label>
                  <input
                    type="text"
                    className="input input-bordered w-full text-black"
                    placeholder="请输入用户名"
                  />
                </div>
                <div>
                  <label className="block text-sm mb-1">密码</label>
                  <input
                    type="password"
                    className="input input-bordered w-full text-black"
                    placeholder="请输入密码"
                  />
                </div>
                <button type="submit" className="btn btn-neutral primary-content w-full">
                  登录
                </button>
              </form>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}
