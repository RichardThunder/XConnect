# macOS .pkg 安装器

CI 中会创建 root/usr/local/bin/ 并复制 xconnect、xconnect-cli、xconnect-tray，
然后执行：

  pkgbuild --root root --identifier com.xconnect.bin --version $VERSION \
    --scripts scripts --install-location / xconnect-$VERSION.pkg

安装后文件位于 /usr/local/bin/。
