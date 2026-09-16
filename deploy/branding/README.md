# 迟早出片品牌包

基于现有站点外观接口配置品牌名、Logo、网页标题和版权署名。保留应用当前配色、布局、登录页媒体与主文案。

- `brand.json`：只包含本产品覆盖的外观字段。
- `logo-light.png` / `logo-dark.png`：原创“片”字几何标记，分别用于明暗界面；同一标记用于 favicon。
- `render-assets.py`：使用 Pillow 重新生成 PNG。

发布时通过管理员外观接口分别上传两个 Logo，再将资源 ID 与 `brand.json` 一起 PATCH 到 `/api/admin/settings/appearance`。先将现有配置保存到受限运维目录；账号密码不进入品牌包。

```sh
python3 deploy/branding/apply-brand.py \
  --base-url https://movie.iaigc.fun \
  --credentials-file .local/deployment/admin-credentials.json \
  --backup-dir .local/deployment/brand-backups
```

品牌包与社区业务代码分离。更新社区源码后复用数据库外观配置和资源卷即可保留品牌；许可证、上游作者署名和插件协议标识继续保留。
