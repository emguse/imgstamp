package main

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"github.com/emguse/imgstamp/internal/cli"
)

var version = "dev"

func main() {
	application := app.NewWithID("com.emguse.imgstamp")
	window := application.NewWindow("imgstamp")
	window.Resize(fyne.NewSize(720, 370))

	configPath := widget.NewEntry()
	configPath.SetText("./stamp.toml")
	inputPath := widget.NewEntry()
	outputPath := widget.NewEntry()
	textOverride := widget.NewEntry()
	textOverride.SetPlaceHolder("設定値と同じ文字数。空欄なら設定値を使用")
	shrink := widget.NewCheck("A1/A2 → A3、A3 → A4 に縮小", nil)
	status := widget.NewLabel("入力・出力フォルダを選択してください")
	log := widget.NewMultiLineEntry()
	log.SetPlaceHolder("処理結果がここに表示されます")
	log.Disable()
	runButton := widget.NewButton("変換", nil)

	configBrowse := widget.NewButton("参照…", func() {
		picker := dialog.NewFileOpen(func(file fyne.URIReadCloser, err error) {
			if err != nil {
				status.SetText("設定ファイルを選択できません: " + err.Error())
				return
			}
			if file == nil {
				return
			}
			defer file.Close()
			configPath.SetText(file.URI().Path())
		}, window)
		picker.SetFilter(storage.NewExtensionFileFilter([]string{".toml"}))
		picker.Show()
	})
	inputBrowse := folderButton(window, inputPath, status, "入力フォルダを選択できません")
	outputBrowse := folderButton(window, outputPath, status, "出力フォルダを選択できません")

	form := container.NewGridWithColumns(3,
		widget.NewLabel("設定ファイル"), configPath, configBrowse,
		widget.NewLabel("入力フォルダ"), inputPath, inputBrowse,
		widget.NewLabel("出力フォルダ"), outputPath, outputBrowse,
		widget.NewLabel("追記テキスト"), textOverride, widget.NewLabel("任意"),
		widget.NewLabel("縮小モード"), shrink, widget.NewLabel(""),
	)

	var running bool
	var cancel context.CancelFunc
	closeAfterCancel := false
	runButton.OnTapped = func() {
		if running {
			return
		}
		args := buildArgs(configPath.Text, inputPath.Text, outputPath.Text, textOverride.Text, shrink.Checked)
		if strings.TrimSpace(configPath.Text) == "" || strings.TrimSpace(inputPath.Text) == "" || strings.TrimSpace(outputPath.Text) == "" {
			status.SetText("設定ファイル、入力フォルダ、出力フォルダを指定してください")
			return
		}
		ctx, stop := context.WithCancel(context.Background())
		cancel = stop
		running = true
		runButton.Disable()
		status.SetText("処理中です…")
		log.SetText("")
		go func() {
			var stdout, stderr bytes.Buffer
			code := cli.Run(ctx, args, &stdout, &stderr, version)
			stop()
			output := strings.TrimSpace(strings.Join(nonempty(stdout.String(), stderr.String()), "\n"))
			fyne.Do(func() {
				running = false
				cancel = nil
				runButton.Enable()
				if output == "" {
					output = fmt.Sprintf("終了コード: %d", code)
				}
				log.SetText(output)
				status.SetText(fmt.Sprintf("終了コード: %d", code))
				if closeAfterCancel {
					application.Quit()
				}
			})
		}()
	}

	window.SetCloseIntercept(func() {
		if !running {
			application.Quit()
			return
		}
		dialog.ShowConfirm("処理中です", "中断してウィンドウを閉じますか？", func(close bool) {
			if close && cancel != nil {
				closeAfterCancel = true
				status.SetText("中断しています…")
				cancel()
			}
		}, window)
	})

	window.SetContent(container.NewBorder(
		container.NewVBox(form, runButton, status),
		nil, nil, nil,
		log,
	))
	window.ShowAndRun()
}

func folderButton(window fyne.Window, target *widget.Entry, status *widget.Label, errorPrefix string) *widget.Button {
	return widget.NewButton("参照…", func() {
		dialog.ShowFolderOpen(func(folder fyne.ListableURI, err error) {
			if err != nil {
				status.SetText(errorPrefix + ": " + err.Error())
				return
			}
			if folder != nil {
				target.SetText(folder.Path())
			}
		}, window)
	})
}

func buildArgs(configPath, inputPath, outputPath, text string, shrink bool) []string {
	args := []string{"--config", configPath, "--input", inputPath, "--output", outputPath}
	if shrink {
		args = append(args, "--shrink")
	}
	if text != "" {
		args = append(args, "--text", text)
	}
	return args
}

func nonempty(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, value)
		}
	}
	return result
}
