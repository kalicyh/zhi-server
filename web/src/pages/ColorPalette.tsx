// File: web/src/pages/ColorPalette.tsx
import React from 'react';

const colors = [
  { name: 'primary', css: 'bg-primary text-primary-content' },
  { name: 'secondary', css: 'bg-secondary text-secondary-content' },
  { name: 'accent', css: 'bg-accent text-accent-content' },
  { name: 'neutral', css: 'bg-neutral text-neutral-content' },
  { name: 'base-100', css: 'bg-base-100 text-base-content' },
  { name: 'base-200', css: 'bg-base-200 text-base-content' },
  { name: 'base-300', css: 'bg-base-300 text-base-content' },
  { name: 'info', css: 'bg-info text-info-content' },
  { name: 'success', css: 'bg-success text-success-content' },
  { name: 'warning', css: 'bg-warning text-warning-content' },
  { name: 'error', css: 'bg-error text-error-content' },
  { name: 'glass', css: 'glass bg-error p-6 backdrop-blur-md bg-opacity-20' },
];

const themeVariables = [
  { variable: '--color-primary', description: '主要品牌颜色' },
  { variable: '--color-primary-content', description: '用于原色的前景内容颜色' },
  { variable: '--color-secondary', description: '次要品牌颜色' },
  { variable: '--color-secondary-content', description: '用于辅助颜色的前景内容颜色' },
  { variable: '--color-accent', description: '强调品牌颜色' },
  { variable: '--color-accent-content', description: '用于强调色的前景内容颜色' },
  { variable: '--color-neutral', description: '中性深色' },
  { variable: '--color-neutral-content', description: '中性色上使用的前景内容颜色' },
  { variable: '--color-base-100', description: '页面基色，用于空白背景' },
  { variable: '--color-base-200', description: '底色，深色' },
  { variable: '--color-base-300', description: '底色，更深的色调' },
  { variable: '--color-base-content', description: '用于基色的前景内容颜色' },
  { variable: '--color-info', description: '信息颜色' },
  { variable: '--color-info-content', description: '信息颜色使用的前景内容颜色' },
  { variable: '--color-success', description: '成功颜色' },
  { variable: '--color-success-content', description: '成功时使用的前景内容颜色' },
  { variable: '--color-warning', description: '警告颜色' },
  { variable: '--color-warning-content', description: '用于警告颜色的前景内容颜色' },
  { variable: '--color-error', description: '错误颜色' },
  { variable: '--color-error-content', description: '错误颜色使用的前景内容颜色' },
  { variable: '--radius-selector', description: '复选框、切换按钮、徽章等选择器的边框半径' },
  { variable: '--radius-field', description: '输入、选择、选项卡等字段的边框半径' },
  { variable: '--radius-box', description: '卡片、模式框、警报框等框的边框半径' },
  { variable: '--size-selector', description: '复选框、切换按钮、徽章等选择器的基本比例尺寸' },
  { variable: '--size-field', description: '输入、选择、选项卡等字段的基本比例尺寸' },
  { variable: '--border', description: '所有组件的边框宽度' },
  { variable: '--depth', description: '（二进制）为相关组件添加深度效果' },
  { variable: '--noise', description: '（二进制）为相关组件添加背景噪声效果' },
];

export default function ColorPalette() {
  return (
    <div className="p-6 bg-base-100">
      <h2 className="text-2xl font-semibold mb-4">daisyUI Color Palette</h2>
      <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
        {colors.map(({ name, css }) => (
          <div key={name} className="rounded-lg overflow-hidden shadow-lg bg-base-100">
            <div className={`${css} p-6 flex flex-col justify-center items-center`}>
              <span className="font-bold capitalize">{name}</span>
            </div>
            <div className="bg-base-100 p-4 text-sm">
              <code className="block">class=&quot;{css}&quot;</code>
            </div>
          </div>
        ))}
      </div>

      <section className="mt-12">
        <h2 className="text-2xl font-semibold mb-4">主题 CSS 变量</h2>
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-base-200">
              <tr>
                <th className="px-4 py-2 text-left text-sm font-medium">CSS 变量</th>
                <th className="px-4 py-2 text-left text-sm font-medium">描述</th>
              </tr>
            </thead>
            <tbody className="bg-base-200 divide-y divide-gray-200">
              {themeVariables.map(({ variable, description }) => (
                <tr key={variable}>
                  <td className="px-4 py-2 text-sm font-mono">{variable}</td>
                  <td className="px-4 py-2 text-sm">{description}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
