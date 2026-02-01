// 托盘应用：系统托盘图标，点击显示主窗口，主窗口展示剪贴板历史（内容 + 来源机器）。
// 需先启动 xconnect 服务（默认 localhost:8315）。
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

const defaultAPIBase = "http://127.0.0.1:8315"

type clipboardEntry struct {
	Content  string    `json:"content"`
	FromHost string    `json:"from_host"`
	At       time.Time `json:"at"`
}

func main() {
	apiBase := os.Getenv("XCONNECT_API")
	if apiBase == "" {
		apiBase = defaultAPIBase
	}

	a := app.New()
	w := a.NewWindow("XConnect 剪贴板历史")
	w.Resize(fyne.NewSize(520, 400))

	var historyEntries []clipboardEntry
	list := widget.NewList(
		func() int { return len(historyEntries) },
		func() fyne.CanvasObject {
			from := widget.NewLabel("")
			from.Wrapping = fyne.TextWrapWord
			content := widget.NewLabel("")
			content.Wrapping = fyne.TextWrapWord
			return container.NewBorder(from, nil, nil, nil, content)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(historyEntries) {
				return
			}
			e := historyEntries[id]
			border := obj.(*fyne.Container)
			top := border.Objects[0].(*widget.Label)     // top
			center := border.Objects[4].(*widget.Label) // center
			top.SetText(fmt.Sprintf("来自: %s  ·  %s", e.FromHost, e.At.Format("15:04:05")))
			preview := e.Content
			if len(preview) > 200 {
				preview = preview[:200] + "…"
			}
			center.SetText(preview)
		},
	)
	status := widget.NewLabel("点击「刷新」从服务拉取历史")
	status.Wrapping = fyne.TextWrapWord

	refresh := func() {
		status.SetText("正在加载…")
		entries, err := fetchHistory(apiBase)
		if err != nil {
			status.SetText("加载失败: " + err.Error())
			list.Refresh()
			return
		}
		historyEntries = entries
		list.Refresh()
		status.SetText(fmt.Sprintf("已加载 %d 条记录", len(historyEntries)))
	}
	refresh()

	bar := container.NewBorder(nil, nil, nil, widget.NewButton("刷新", refresh), status)
	content := container.NewBorder(bar, nil, nil, nil, list)
	w.SetContent(content)

	w.SetCloseIntercept(func() {
		w.Hide()
	})

	if desk, ok := a.(desktop.App); ok {
		m := fyne.NewMenu("XConnect",
			fyne.NewMenuItem("显示主窗口", func() { w.Show() }),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("退出", func() { a.Quit() }),
		)
		desk.SetSystemTrayMenu(m)
	}

	w.ShowAndRun()
}

func fetchHistory(apiBase string) ([]clipboardEntry, error) {
	url := apiBase + "/clipboard/history"
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %s", resp.Status)
	}
	var entries []clipboardEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, err
	}
	return entries, nil
}
