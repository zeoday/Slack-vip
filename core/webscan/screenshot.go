package webscan

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"slack-wails/lib/utils"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

var (
	screenshotDir = filepath.Join(utils.HomeDir(), "slack", "screenshot")
	// 全局 Chrome Allocator（只初始化一次）
	allocCtx  context.Context
	allocOnce sync.Once
)

func init() {
	// 创建截屏文件服务器
	go func() {
		fs := http.FileServer(http.Dir(screenshotDir))

		// 创建独立的 ServeMux
		mux := http.NewServeMux()
		mux.Handle("/screenhost/", http.StripPrefix("/screenhost", fs))

		// 启动 HTTP 服务器
		err := http.ListenAndServe(":8732", mux)
		if err != nil {
			return
		}
	}()

	allocOnce.Do(func() {
		opts := append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true),
			chromedp.Flag("disable-gpu", true),

			// 🔐 HTTPS / 扫描目标必备
			chromedp.Flag("ignore-certificate-errors", true),
			chromedp.Flag("allow-insecure-localhost", true),
			chromedp.Flag("disable-web-security", true),

			// 🚫 减少后台资源占用
			chromedp.Flag("disable-background-networking", true),
			chromedp.Flag("disable-background-timer-throttling", true),
			chromedp.Flag("disable-backgrounding-occluded-windows", true),
			chromedp.Flag("disable-renderer-backgrounding", true),
		)

		allocCtx, _ = chromedp.NewExecAllocator(context.Background(), opts...)
	})
}

// GetScreenshot 获取指定URL的屏幕截图，并保存到本地文件。
// 返回文件路径和错误，如果错误不为nil，则文件路径为空。
func GetScreenshot(url string) (string, error) {
	if err := os.MkdirAll(screenshotDir, 0755); err != nil {
		return "", err
	}

	filename := utils.RenameOutput(url) + ".png"
	relativePath := filepath.Join(screenshotDir, filename)

	if _, err := os.Stat(relativePath); err == nil {
		return relativePath, nil
	}

	// 每次截图 = 新 tab（不是新 Chrome）
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// ⏱️ 强制超时（防止 HTTPS / JS 卡死）
	ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var buf []byte

	err := chromedp.Run(ctx,
		// 固定视口，避免超大截图
		chromedp.EmulateViewport(1366, 768),

		// 导航
		chromedp.Navigate(url),

		// 等待页面稳定（比 Sleep 靠谱）
		chromedp.WaitReady("body", chromedp.ByQuery),

		// 截图（非 FullScreenshot，内存安全）
		chromedp.CaptureScreenshot(&buf),
	)
	if err != nil {
		return "", errors.New("截图失败: " + err.Error())
	}

	if err := os.WriteFile(relativePath, buf, 0644); err != nil {
		return "", err
	}

	return relativePath, nil
}
