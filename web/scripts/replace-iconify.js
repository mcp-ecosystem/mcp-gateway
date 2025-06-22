#!/usr/bin/env node

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const srcDir = path.join(__dirname, '..', 'src');

// 需要替换的文件列表
const filesToReplace = [
  'pages/auth/login.tsx',
  'components/LanguageSwitcher.tsx',
  'pages/users/user-management.tsx',
  'pages/users/tenant-management.tsx',
  'components/ChangePasswordDialog.tsx',
  'components/ui/MultiSelectAutocomplete.tsx',
  'components/Layout.tsx',
  'pages/gateway/gateway-manager.tsx',
  'pages/chat/chat-interface.tsx',
  'pages/gateway/components/RouterConfig.tsx',
  'pages/chat/components/chat-history.tsx',
  'pages/gateway/components/MCPServersConfig.tsx',
  'pages/gateway/components/ToolsConfig.tsx',
  'pages/gateway/components/ServersConfig.tsx',
  'pages/gateway/config-versions.tsx',
  'pages/gateway/components/OpenAPIImport.tsx',
  'pages/chat/components/chat-message.tsx'
];

function replaceInFile(filePath) {
  try {
    let content = fs.readFileSync(filePath, 'utf8');
    let hasChanges = false;

    // 替换导入语句
    const importRegex = /import\s+{\s*Icon\s*}\s+from\s+['"]@iconify\/react['"];?\n?/g;
    if (importRegex.test(content)) {
      content = content.replace(importRegex, "import LocalIcon from '../../../components/LocalIcon';\n");
      hasChanges = true;
    }

    // 替换相对路径的导入（根据文件位置调整）
    const levels = filePath.split('/').length - srcDir.split('/').length - 1;
    const relativePath = '../'.repeat(levels) + 'components/LocalIcon';
    
    content = content.replace(
      /import\s+{\s*Icon\s*}\s+from\s+['"]@iconify\/react['"];?\n?/g, 
      `import LocalIcon from '${relativePath}';\n`
    );

    // 替换组件使用
    const iconUsageRegex = /<Icon\s+icon=/g;
    if (iconUsageRegex.test(content)) {
      content = content.replace(iconUsageRegex, '<LocalIcon icon=');
      hasChanges = true;
    }

    if (hasChanges) {
      fs.writeFileSync(filePath, content);
      console.log(`✓ 已更新: ${filePath}`);
      return true;
    } else {
      console.log(`- 无变化: ${filePath}`);
      return false;
    }
  } catch (error) {
    console.error(`✗ 错误: ${filePath}`, error.message);
    return false;
  }
}

function main() {
  console.log('开始替换 @iconify/react 导入...\n');
  
  let updatedCount = 0;
  
  for (const file of filesToReplace) {
    const fullPath = path.join(srcDir, file);
    if (fs.existsSync(fullPath)) {
      if (replaceInFile(fullPath)) {
        updatedCount++;
      }
    } else {
      console.log(`! 文件不存在: ${fullPath}`);
    }
  }
  
  console.log(`\n完成！共更新了 ${updatedCount} 个文件。`);
}

main();