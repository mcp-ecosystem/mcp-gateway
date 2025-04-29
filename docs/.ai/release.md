# Release Notes

每次发布新版本时，按照以下进行操作：
1. 可以通过`cat pkg/version/VERSION`
2. 请参考changelog/`cat pkg/version/VERSION`.md文件的书写风格和格式（例如标准开头，然后先中文后英文）
3. 通过git命令去查找从前一个版本到现在的变更，如`git log v0.2.6..HEAD --pretty=format:"%h %s" | cat`
4. 改变pkg/version/VERSION和web/package.json（更新后最好cd web && npm i一下）中的版本号，如果没有特别指明，通常是+0.0.1
5. 根据2和3的内容，新增新版本的变更内容到changelog/`cat pkg/version/VERSION`.md