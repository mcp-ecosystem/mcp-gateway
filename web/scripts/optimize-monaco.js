#!/usr/bin/env node

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const monacoDir = path.join(__dirname, '..', 'public', 'monaco-editor', 'vs');

// 只保留项目需要的文件
const keepFiles = [
  // 核心文件
  'loader.js',
  
  // 基础文件
  'base/worker/workerMain.js',
  
  // 编辑器核心
  'editor/editor.main.js',
  'editor/editor.main.css',
  
  // YAML语言支持
  'basic-languages/yaml/yaml.js',
  
  // 如果需要JSON支持可以保留，但项目只用YAML
  // 'language/json/jsonWorker.js',
  // 'language/json/jsonMode.js',
  
  // 只保留中英文语言包
  'nls.messages.zh-cn.js',
  // 可以保留英文作为默认，但通常loader.js已包含
];

// 要删除的大文件夹
const deleteDirs = [
  'language/typescript',  // 5.5M - 不需要TS支持
  'language/html',        // 476K - 不需要HTML支持  
  'language/css',         // 796K - 不需要CSS支持
  'language/json',        // 176K - 不需要JSON支持，只用YAML
];

// 要删除的语言包（保留中文，删除其他）
const deleteLanguages = [
  'nls.messages.de.js',
  'nls.messages.es.js', 
  'nls.messages.fr.js',
  'nls.messages.it.js',
  'nls.messages.ja.js',
  'nls.messages.ko.js',
  'nls.messages.ru.js',
  'nls.messages.zh-tw.js',
];

// 删除不需要的basic-languages（保留yaml，删除其他）
const basicLanguagesDir = path.join(monacoDir, 'basic-languages');
const keepBasicLanguages = ['yaml'];

function deleteRecursive(dirPath) {
  if (fs.existsSync(dirPath)) {
    if (fs.statSync(dirPath).isDirectory()) {
      fs.rmSync(dirPath, { recursive: true, force: true });
      console.log(`✓ 删除目录: ${path.relative(monacoDir, dirPath)}`);
    } else {
      fs.unlinkSync(dirPath);
      console.log(`✓ 删除文件: ${path.relative(monacoDir, dirPath)}`);
    }
  }
}

function optimizeMonaco() {
  console.log('开始优化Monaco Editor...\n');
  
  let savedSpace = 0;
  
  // 删除大的语言支持目录
  for (const dir of deleteDirs) {
    const fullPath = path.join(monacoDir, dir);
    if (fs.existsSync(fullPath)) {
      const stats = fs.statSync(fullPath);
      const size = getDirectorySize(fullPath);
      savedSpace += size;
      deleteRecursive(fullPath);
    }
  }
  
  // 删除不需要的语言包
  for (const lang of deleteLanguages) {
    const fullPath = path.join(monacoDir, lang);
    if (fs.existsSync(fullPath)) {
      const stats = fs.statSync(fullPath);
      savedSpace += stats.size;
      deleteRecursive(fullPath);
    }
  }
  
  // 清理basic-languages目录，只保留yaml
  if (fs.existsSync(basicLanguagesDir)) {
    const languages = fs.readdirSync(basicLanguagesDir);
    for (const lang of languages) {
      if (!keepBasicLanguages.includes(lang)) {
        const langPath = path.join(basicLanguagesDir, lang);
        const size = getDirectorySize(langPath);
        savedSpace += size;
        deleteRecursive(langPath);
      }
    }
  }
  
  console.log(`\n优化完成！`);
  console.log(`节省空间: ${(savedSpace / 1024 / 1024).toFixed(1)} MB`);
  
  // 显示剩余大小
  const remainingSize = getDirectorySize(monacoDir);
  console.log(`剩余大小: ${(remainingSize / 1024 / 1024).toFixed(1)} MB`);
}

function getDirectorySize(dirPath) {
  let size = 0;
  
  function calculateSize(itemPath) {
    const stats = fs.statSync(itemPath);
    if (stats.isDirectory()) {
      const items = fs.readdirSync(itemPath);
      for (const item of items) {
        calculateSize(path.join(itemPath, item));
      }
    } else {
      size += stats.size;
    }
  }
  
  if (fs.existsSync(dirPath)) {
    calculateSize(dirPath);
  }
  
  return size;
}

optimizeMonaco();