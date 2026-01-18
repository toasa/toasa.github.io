package main

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// 設定：出力先ディレクトリ
const outputDir = "public/diary"
const dataDir = "data"

type DayEntry struct {
	Year    string
	Month   string
	Day     string
	Content template.HTML
	Path    string
}

type MonthSection struct {
	Month string
	Days  []DayEntry
}

type YearSection struct {
	Year   string
	Months []*MonthSection
}

func main() {
	// 1. データの収集
	entries, err := collectEntries(dataDir)
	if err != nil {
		panic(err)
	}

	// 2. 出力ディレクトリのクリーニングと作成
	os.RemoveAll("public")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		panic(err)
	}

	// 3. 階層構造データの作成
	years := groupEntriesByDate(entries)

	// 4. HTMLの生成

	// 4-1. 各日のページ
	for _, entry := range entries {
		html := renderTemplate("day", entry)
		saveFile(filepath.Join(outputDir, entry.Year, entry.Month, entry.Day+".html"), html)
	}

	// 4-2. 月のページ
	for _, y := range years {
		for _, m := range y.Months {
			data := map[string]interface{}{
				"Year": y.Year, "Month": m.Month, "Days": m.Days,
			}
			html := renderTemplate("month", data)
			saveFile(filepath.Join(outputDir, y.Year, m.Month+".html"), html)
		}
	}

	// 4-3. 年のページ
	for _, y := range years {
		data := map[string]interface{}{
			"Year": y.Year, "Months": y.Months,
		}
		html := renderTemplate("year", data)
		saveFile(filepath.Join(outputDir, y.Year+".html"), html)
	}

	// 4-4. トップページ
	var latestEntry *DayEntry
	if len(entries) > 0 {
		latestEntry = &entries[len(entries)-1]
	}

	data := map[string]interface{}{
		"Years":  years,
		"Latest": latestEntry,
	}
	html := renderTemplate("index", data)
	saveFile(filepath.Join(outputDir, "index.html"), html)

	fmt.Println("生成完了: ./public ディレクトリを確認してください")
}

func collectEntries(root string) ([]DayEntry, error) {
	var entries []DayEntry

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),          // GitHub Flavored Markdown
		goldmark.WithRendererOptions(html.WithUnsafe()), // WithUnsafe: <img>タグなどをそのまま出力する
	)

	startDate := time.Date(2022, 8, 11, 0, 0, 0, 0, time.Local)
	today := time.Now()

	for d := startDate; !d.After(today); d = d.AddDate(0, 0, 1) {
		year := fmt.Sprintf("%04d", d.Year())
		month := fmt.Sprintf("%02d", int(d.Month()))
		day := fmt.Sprintf("%02d", d.Day())

		path := filepath.Join(root, year, month, day+".md")

		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				continue // ファイルがない日はスキップ
			}
			return nil, err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		var buf bytes.Buffer
		if err := md.Convert(content, &buf); err != nil {
			return nil, err
		}

		entries = append(entries, DayEntry{
			Year:    year,
			Month:   month,
			Day:     day,
			Content: template.HTML(buf.String()),
		})
	}

	return entries, nil
}

func groupEntriesByDate(entries []DayEntry) []*YearSection {
	var years []*YearSection

	for _, e := range entries {
		// 年の処理: 最後の年と違う、またはまだない場合は新しい年を追加
		if len(years) == 0 || years[len(years)-1].Year != e.Year {
			years = append(years, &YearSection{
				Year: e.Year,
			})
		}
		currentYear := years[len(years)-1]

		// 月の処理: その年の最後の月と違う、またはまだない場合は新しい月を追加
		if len(currentYear.Months) == 0 || currentYear.Months[len(currentYear.Months)-1].Month != e.Month {
			currentYear.Months = append(currentYear.Months, &MonthSection{
				Month: e.Month,
			})
		}
		currentMonth := currentYear.Months[len(currentYear.Months)-1]

		// 日の追加
		currentMonth.Days = append(currentMonth.Days, e)
	}

	return years
}

func saveFile(path string, content string) {
	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0755)
	os.WriteFile(path, []byte(content), 0644)
}

func renderTemplate(kind string, data interface{}) string {
	const tplString = `<!DOCTYPE html>
<html>

<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>日記</title>
    <style>
        body {
            font-family: sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            line-height: 1.6;
            /* color: #333; */
            color: #1f2328;
            /* GitHub風の少し濃い黒 */
        }

        /* スマホなど画面が狭い時の余白調整 */
        @media (max-width: 600px) {
            body {
                padding: 15px;
            }
        }

        img {
            max-width: 100%;
            height: auto;
        }

        a {
            color: #0066cc;
        }

        code {
            background-color: rgba(175, 184, 193, 0.2);
            padding: 0.2em 0.4em;
            font-size: 85%;
            border-radius: 6px;
            font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, "Liberation Mono", monospace;
        }

        pre {
            background-color: #f6f8fa;
            padding: 16px;
            border-radius: 6px;
            overflow: auto;
        }

        pre code {
            background-color: transparent;
            padding: 0;
        }

        blockquote {
            border-left: 4px solid #ccc;
            margin: 1em 0;
            padding-left: 16px;
            color: #555;
        }

        /* GitHubスタイルのテーブル設定 */
        table {
            border-spacing: 0;
            border-collapse: collapse;
            margin-top: 0;
            margin-bottom: 16px;
            width: 100%;
            display: block;
            overflow: auto;
        }

        table th {
            font-weight: 600;
        }

        table th,
        table td {
            padding: 6px 13px;
            border: 1px solid #d0d7de;
        }

        table tr {
            background-color: #ffffff;
            border-top: 1px solid #d8dee4;
        }

        /* 1行おきに背景色を変える（ゼブラ縞） */
        table tr:nth-child(2n) {
            background-color: #f6f8fa;
        }

        nav {
            margin-bottom: 20px;
        }

        /* hr { border: 0; border-top: 1px solid #eee; margin: 20px 0; } */
    </style>
</head>

<body>
    {{if ne .Kind "index"}}
        <nav>
            <a href="/diary/index.html">トップ</a>
            {{if eq .Kind "month"}}
                / <a href="/diary/{{.Data.Year}}.html">{{.Data.Year}}</a>
            {{else if eq .Kind "day"}}
                / <a href="/diary/{{.Data.Year}}.html">{{.Data.Year}}</a>
                / <a href="/diary/{{.Data.Year}}/{{.Data.Month}}.html">{{.Data.Month}}</a>
            {{end}}
        </nav>
        <hr>
    {{end}}

    {{if eq .Kind "index"}}
        <h2>日記</h2>

        {{if .Data.Latest}}
            <h3>最新</h3>
            <a href="{{.Data.Latest.Year}}/{{.Data.Latest.Month}}/{{.Data.Latest.Day}}.html">
                {{.Data.Latest.Year}}年{{.Data.Latest.Month}}月{{.Data.Latest.Day}}日
            </a>
        {{end}}

        <h3>アーカイブ</h3>
        <ul>
            {{range .Data.Years}}
                <li><a href="{{.Year}}.html">{{.Year}}年</a></li>
            {{end}}
        </ul>

    {{else if eq .Kind "year"}}
        <h2>{{.Data.Year}}</h2>
        <ul>
            {{range .Data.Months}}
                <li><a href="{{$.Data.Year}}/{{.Month}}.html">{{.Month}}月</a></li>
            {{end}}
        </ul>

    {{else if eq .Kind "month"}}
        <h2>{{.Data.Year}}/{{.Data.Month}}</h2>
        <ul>
            {{range .Data.Days}}
                <li><a href="{{.Month}}/{{.Day}}.html">{{.Day}}日</a></li>
            {{end}}
        </ul>

    {{else if eq .Kind "day"}}
        <h2>{{.Data.Year}}/{{.Data.Month}}/{{.Data.Day}}</h2>
        <div>{{.Data.Content}}</div>

    {{end}}
</body>

</html>`

	t, _ := template.New("base").Parse(tplString)
	var buf bytes.Buffer
	wrapper := struct {
		Kind string
		Data interface{}
	}{Kind: kind, Data: data}

	t.Execute(&buf, wrapper)
	return buf.String()
}
