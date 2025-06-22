#!/usr/bin/env node

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// 所有需要替换的文件及其相对路径深度
const files = [
  { path: 'src/pages/auth/login.tsx', depth: 3 },
  { path: 'src/components/LanguageSwitcher.tsx', depth: 1 },
  { path: 'src/pages/users/user-management.tsx', depth: 3 },
  { path: 'src/pages/users/tenant-management.tsx', depth: 3 },
  { path: 'src/components/ChangePasswordDialog.tsx', depth: 1 },
  { path: 'src/components/ui/MultiSelectAutocomplete.tsx', depth: 2 },
  { path: 'src/components/Layout.tsx', depth: 1 },
  { path: 'src/pages/gateway/gateway-manager.tsx', depth: 3 },
  { path: 'src/pages/chat/chat-interface.tsx', depth: 3 },
  { path: 'src/pages/gateway/components/RouterConfig.tsx', depth: 4 },
  { path: 'src/pages/chat/components/chat-history.tsx', depth: 4 },
  { path: 'src/pages/gateway/components/MCPServersConfig.tsx', depth: 4 },
  { path: 'src/pages/gateway/components/ToolsConfig.tsx', depth: 4 },
  { path: 'src/pages/gateway/components/ServersConfig.tsx', depth: 4 },
  { path: 'src/pages/gateway/config-versions.tsx', depth: 3 },
  { path: 'src/pages/gateway/components/OpenAPIImport.tsx', depth: 4 },
  { path: 'src/pages/chat/components/chat-message.tsx', depth: 4 }
];

function replaceInFile(fileInfo) {
  const fullPath = path.join(__dirname, '..', fileInfo.path);
  
  if (!fs.existsSync(fullPath)) {
    console.log(`! 文件不存在: ${fileInfo.path}`);
    return false;
  }

  try {
    let content = fs.readFileSync(fullPath, 'utf8');
    let hasChanges = false;
    
    // 计算相对路径
    const relativePath = '../'.repeat(fileInfo.depth) + 'components/LocalIcon';

    // 替换导入语句
    const importRegex = /import\s+{\s*Icon\s*}\s+from\s+['"]@iconify\/react['"];?\n?/g;
    if (content.match(importRegex)) {
      content = content.replace(importRegex, `import LocalIcon from '${relativePath}';\n`);
      hasChanges = true;
    }

    // 替换所有 <Icon 为 <LocalIcon
    const iconRegex = /<Icon\s+/g;
    if (content.match(iconRegex)) {
      content = content.replace(iconRegex, '<LocalIcon ');
      hasChanges = true;
    }

    if (hasChanges) {
      fs.writeFileSync(fullPath, content);
      console.log(`✓ 已更新: ${fileInfo.path}`);
      return true;
    } else {
      console.log(`- 无变化: ${fileInfo.path}`);
      return false;
    }
  } catch (error) {
    console.error(`✗ 错误: ${fileInfo.path}`, error.message);
    return false;
  }
}

function main() {
  console.log('开始批量替换 @iconify/react...\n');
  
  let updatedCount = 0;
  
  for (const fileInfo of files) {
    // 跳过已经处理过的LanguageSwitcher.tsx
    if (fileInfo.path.includes('LanguageSwitcher.tsx')) {
      console.log(`- 已处理: ${fileInfo.path}`);
      continue;
    }
    
    if (replaceInFile(fileInfo)) {
      updatedCount++;
    }
  }
  
  console.log(`\n完成！共更新了 ${updatedCount} 个文件。`);
}

main();