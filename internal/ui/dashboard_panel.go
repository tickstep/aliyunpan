// Copyright (c) 2020 tickstep.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package ui

import (
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/tickstep/aliyunpan/internal/utils"

	"github.com/tickstep/library-go/converter"
	"github.com/tickstep/library-go/requester/rio/speeds"
)

type TaskState int
type DashboardType int

const (
	TaskQueued TaskState = iota
	TaskRunning
	TaskSuccess
	TaskFailed
	TaskSkipped
	TaskCanceled

	// DashboardPanelDownload 下载面板类型
	DashboardPanelDownload DashboardType = 1
	// DashboardPanelUpload 上传面板类型
	DashboardPanelUpload DashboardType = 2
)

type DashboardOptions struct {
	Title       string
	MaxHistory  int
	ActiveSlots int
	Output      io.Writer
}

// DashboardPanel 统计面板使用了很多ANSI转义序列。并不是所有的终端都支持ANSI转义序列，所以需要做兼容性处理。
type DashboardPanel struct {
	dashboardType DashboardType
	mu            sync.Mutex
	tasks         map[string]*dashboardTask
	order         []string
	history       []string
	maxHistory    int
	parallel      int
	activeSlots   int
	globalSpeeds  *speeds.Speeds
	title         string
	startTime     time.Time
	renderEvery   time.Duration
	stopCh        chan struct{}
	doneCh        chan struct{}
	startOnce     sync.Once
	closeOnce     sync.Once
	out           io.Writer
}

type dashboardTask struct {
	id         string
	path       string
	name       string
	total      int64
	downloaded int64
	speed      int64
	eta        time.Duration
	state      TaskState
	message    string
	isFile     bool
}

// NewDashboardPanel 创建下载任务统计面板
func NewDashboardPanel(dashboardType DashboardType, parallel int, globalSpeeds *speeds.Speeds, opts *DashboardOptions) *DashboardPanel {
	if parallel < 1 {
		parallel = 1
	}
	db := &DashboardPanel{
		dashboardType: dashboardType,
		tasks:         make(map[string]*dashboardTask),
		order:         make([]string, 0, 128),
		history:       make([]string, 0, 64),
		maxHistory:    8,
		parallel:      parallel,
		activeSlots:   5,
		globalSpeeds:  globalSpeeds,
		title:         "aliyunpan统计面板",
		renderEvery:   500 * time.Millisecond,
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
		out:           os.Stdout,
	}
	if opts != nil {
		if opts.Title != "" {
			db.title = opts.Title
		}
		if opts.MaxHistory > 0 {
			db.maxHistory = opts.MaxHistory
		}
		if opts.ActiveSlots > 0 {
			db.activeSlots = opts.ActiveSlots
		}
		if opts.Output != nil {
			db.out = opts.Output
		}
	}
	if db.activeSlots < 1 {
		db.activeSlots = 1
	}
	return db
}

// Start 启动面板
func (db *DashboardPanel) Start() {
	if db == nil {
		return
	}
	db.startOnce.Do(func() {
		db.startTime = time.Now()
		enableVirtualTerminalProcessing()
		fmt.Fprint(db.out, "\x1b[?25l") // 隐藏光标
		db.render()
		ticker := time.NewTicker(db.renderEvery)
		go func() {
			defer close(db.doneCh)
			defer ticker.Stop()
			for {
				select {
				case <-db.stopCh:
					// 延迟1秒
					time.Sleep(1 * time.Second)
					// 渲染
					db.render()
					return
				case <-ticker.C:
					db.render()
				}
			}
		}()
	})
}

// Close 关闭面板
func (db *DashboardPanel) Close() {
	if db == nil {
		return
	}
	db.closeOnce.Do(func() {
		close(db.stopCh)
		<-db.doneCh
		fmt.Fprint(db.out, "\x1b[?25h") // 显示光标
	})
}

// RegisterTask 注册任务信息
func (db *DashboardPanel) RegisterTask(id, path string, total int64, isFile bool) {
	if db == nil || id == "" {
		return
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	task := db.getOrCreateTaskLocked(id)
	if path != "" {
		task.path = path
	}
	if task.name == "" {
		task.name = baseName(path)
	}
	if total > 0 {
		task.total = total
	}
	task.isFile = isFile
}

// UpdateTaskInfo 更新任务信息
func (db *DashboardPanel) UpdateTaskInfo(id, name string, total int64, isFile bool) {
	if db == nil || id == "" {
		return
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	task := db.getOrCreateTaskLocked(id)
	if name != "" {
		task.name = name
	}
	if total > 0 {
		task.total = total
	}
	task.isFile = isFile
}

// UpdateTaskProgress 更新任务进度
func (db *DashboardPanel) UpdateTaskProgress(id string, downloaded, total, speed int64, eta time.Duration) {
	if db == nil || id == "" {
		return
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	task := db.getOrCreateTaskLocked(id)
	task.downloaded = downloaded
	if total > 0 {
		task.total = total
	}
	task.speed = speed
	task.eta = eta
}

// MarkTaskState 标记任务状态
func (db *DashboardPanel) MarkTaskState(id string, state TaskState, message string) {
	if db == nil || id == "" {
		return
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	task := db.getOrCreateTaskLocked(id)
	task.state = state
	if message != "" {
		task.message = message
	}
	switch state {
	case TaskSuccess, TaskFailed, TaskSkipped, TaskCanceled:
		db.appendHistoryLocked(db.defaultHistoryMessage(task, message))
	}
}

// Logf 增加日志
func (db *DashboardPanel) Logf(format string, a ...interface{}) {
	if db == nil {
		return
	}
	msg := fmt.Sprintf(format, a...)
	msg = strings.TrimRight(msg, "\r\n")
	if msg == "" {
		return
	}
	// 去掉开头的换行符
	msg = strings.TrimPrefix(msg, "\n")
	// 去掉回车换行符，UI面板的日志只能单行显示
	msg = strings.ReplaceAll(msg, "\n", " ")
	db.mu.Lock()
	defer db.mu.Unlock()
	db.appendHistoryLocked(msg)
}

func (db *DashboardPanel) getOrCreateTaskLocked(id string) *dashboardTask {
	if task, ok := db.tasks[id]; ok {
		return task
	}
	task := &dashboardTask{
		id:    id,
		state: TaskQueued,
	}
	db.tasks[id] = task
	db.order = append(db.order, id)
	return task
}

func (db *DashboardPanel) appendHistoryLocked(message string) {
	if message == "" {
		return
	}
	timestamp := time.Now().Format("15:04:05")
	db.history = append(db.history, fmt.Sprintf("[%s] %s", timestamp, message))
	if len(db.history) > db.maxHistory*3 {
		db.history = db.history[len(db.history)-db.maxHistory*2:]
	}
}

func (db *DashboardPanel) defaultHistoryMessage(task *dashboardTask, message string) string {
	if message != "" {
		return message
	}
	name := task.path
	if name == "" {
		name = task.name
	}
	switch task.state {
	case TaskSuccess:
		return "完成: " + name
	case TaskFailed:
		return "失败: " + name
	case TaskSkipped:
		return "跳过: " + name
	case TaskCanceled:
		return "取消: " + name
	default:
		return name
	}
}

// render 渲染面板
func (db *DashboardPanel) render() {
	if db == nil {
		return
	}
	width, height := terminalSize()
	if width < 60 {
		width = 60
	}
	innerWidth := width - 2
	snapshot := db.snapshot()

	// 统计总览
	headerLines := db.headerLines(snapshot, innerWidth)

	// 下载任务队列详情
	//activeTitle := db.activeTitle(snapshot)
	activeLines := db.activeLines(snapshot, innerWidth)

	// 下载日志
	headerHeight := len(headerLines) + 2
	activeHeight := len(activeLines) + 1
	historyBase := 3
	minHistory := 1
	minHeight := headerHeight + activeHeight + historyBase + minHistory
	if height < minHeight {
		height = minHeight
	}
	historyLines := height - (headerHeight + activeHeight + historyBase)
	if historyLines < minHistory {
		historyLines = minHistory
	}
	historyBody := db.historyLines(snapshot, innerWidth, historyLines)

	// 拼接并格式化输出
	builder := &strings.Builder{}
	builder.WriteString(renderBox(headerLines, width))
	builder.WriteString(renderTitledBox("ActiveDownloads", activeLines, width))
	builder.WriteString(renderTitledBox("History", historyBody, width))
	content := builder.String()

	// 使用双缓冲技术
	var buf strings.Builder
	// 构建完整内容
	buf.WriteString("\x1b[H\x1b[2J") // 清屏操作
	buf.WriteString(content)
	// 一次性输出
	fmt.Fprint(db.out, buf.String())
}

type dashboardSnapshot struct {
	tasks   []*dashboardTask
	history []string
}

func (db *DashboardPanel) snapshot() dashboardSnapshot {
	db.mu.Lock()
	defer db.mu.Unlock()
	tasks := make([]*dashboardTask, 0, len(db.order))
	for _, id := range db.order {
		if task, ok := db.tasks[id]; ok {
			copied := *task
			tasks = append(tasks, &copied)
		}
	}
	history := append([]string(nil), db.history...)
	return dashboardSnapshot{tasks: tasks, history: history}
}

func renderBox(lines []string, width int) string {
	if width < 4 {
		width = 4
	}
	innerWidth := width - 2
	builder := &strings.Builder{}
	builder.WriteString("\u250c")
	builder.WriteString(strings.Repeat("\u2500", innerWidth))
	builder.WriteString("\u2510")
	for _, line := range lines {
		builder.WriteString("\n")
		builder.WriteString("\u2502") // 左边框
		builder.WriteString(fitLine(line, innerWidth))
		builder.WriteString("\u2502") // 右边框
	}
	builder.WriteString("\n")
	builder.WriteString("\u2502") // 左边框
	//builder.WriteString("\u2514") // 左下角
	builder.WriteString(strings.Repeat("\u2500", innerWidth))
	builder.WriteString("\u2502") // 右边框
	//builder.WriteString("\u2518") // 右下角
	return builder.String()
}

func renderTitledBox(title string, body []string, width int) string {
	if width < 4 {
		width = 4
	}
	innerWidth := width - 2
	builder := &strings.Builder{}

	for _, line := range body {
		builder.WriteString("\n")
		builder.WriteString("\u2502")                  // 左边框
		builder.WriteString(fitLine(line, innerWidth)) // 内容，使用空格补足宽度
		builder.WriteString("\u2502")                  // 右边框
	}

	builder.WriteString("\n")
	if strings.HasPrefix(title, "History") {
		builder.WriteString("\u2514") // 左下角
	} else {
		builder.WriteString("\u2502") // 左边框
	}
	builder.WriteString(strings.Repeat("\u2500", innerWidth)) // 表格横线
	if strings.HasPrefix(title, "History") {
		builder.WriteString("\u2518") // 右下角
	} else {
		builder.WriteString("\u2502") // 右边框
	}
	return builder.String()
}

func (db *DashboardPanel) headerLines(snapshot dashboardSnapshot, innerWidth int) []string {
	totalFiles, doneFiles, failedFiles := db.fileCounts(snapshot.tasks)
	downloaded, total := db.totalProgress(snapshot.tasks)
	speed := db.totalSpeed(snapshot.tasks)
	elapsed := time.Since(db.startTime)
	left := "-"
	if speed > 0 && total > downloaded {
		left = formatDurationShort(time.Duration((total-downloaded)/speed) * time.Second)
	}

	status := "进行中"
	if db.dashboardType == DashboardPanelDownload {
		status = "下载中"
	} else if db.dashboardType == DashboardPanelUpload {
		status = "上传中"
	}
	if totalFiles > 0 && doneFiles >= totalFiles {
		if failedFiles > 0 {
			status = "已完成 (有错误)"
		} else {
			status = "已完成"
		}
	}

	line1 := fmt.Sprintf("总速度: %s | 文件: %d/%d | 失败: %d | 状态: %s | 已用时间: %s | 剩余时间: %s",
		utils.FormatSpeedFixedWidth(speed, 11), doneFiles, totalFiles, failedFiles, status, formatDurationShort(elapsed), left)
	line1 = fitLine(line1, innerWidth)

	progressPct := 0.0
	if total > 0 {
		progressPct = float64(downloaded) / float64(total)
	}
	progress := fmt.Sprintf("%s/%s", converter.ConvertFileSize(downloaded, 2), converter.ConvertFileSize(total, 2))
	label := "总进度: "
	suffix := fmt.Sprintf(" %5.1f%% (%s)", progressPct*100, progress)
	barWidth := innerWidth - displayWidth(label) - displayWidth(suffix) - 2
	if barWidth < 10 {
		barWidth = 10
	}
	bar := renderBar(progressPct, barWidth)
	line2 := fmt.Sprintf("%s[%s]%s", label, bar, suffix)
	line2 = fitLine(line2, innerWidth)

	return []string{line1, line2}
}

func (db *DashboardPanel) activeTitle(snapshot dashboardSnapshot) string {
	running := 0
	queued := 0
	for _, task := range snapshot.tasks {
		if !task.isFile {
			continue
		}
		switch task.state {
		case TaskRunning:
			running++
		case TaskQueued:
			queued++
		}
	}
	return fmt.Sprintf("Active Downloads (slots %d, running %d, queued %d)", db.activeSlots, running, queued)
}

func (db *DashboardPanel) activeLines(snapshot dashboardSnapshot, innerWidth int) []string {
	active := db.pickActiveTasks(snapshot.tasks)
	lines := make([]string, 0, db.activeSlots)
	for i := 0; i < db.activeSlots; i++ {
		if i < len(active) {
			lines = append(lines, renderTaskLine(active[i], i, innerWidth))
			continue
		}
		placeholder := &dashboardTask{name: "[空闲...]", state: TaskQueued}
		lines = append(lines, renderTaskLine(placeholder, i, innerWidth))
	}
	return lines
}

func (db *DashboardPanel) historyLines(snapshot dashboardSnapshot, innerWidth int, lines int) []string {
	if lines < 1 {
		lines = 1
	}
	maxLines := lines
	if db.maxHistory > 0 && maxLines > db.maxHistory {
		maxLines = db.maxHistory
	}
	if maxLines > len(snapshot.history) {
		maxLines = len(snapshot.history)
	}
	result := make([]string, 0, lines)
	start := len(snapshot.history) - maxLines
	if start < 0 {
		start = 0
	}
	for i := len(snapshot.history) - 1; i >= start && len(result) < maxLines; i-- {
		result = append(result, truncateText(snapshot.history[i], innerWidth))
	}
	if len(result) == 0 {
		result = append(result, truncateText("No recent events.", innerWidth))
	}

	// 倒序数组
	reversedResult := make([]string, 0, len(result))
	for i := len(result) - 1; i >= 0; i-- {
		reversedResult = append(reversedResult, result[i])
	}
	result = reversedResult

	// 补全空行
	for len(result) < lines {
		result = append(result, "")
	}
	return result[:lines]
}

func (db *DashboardPanel) fileCounts(tasks []*dashboardTask) (total int, done int, failed int) {
	for _, task := range tasks {
		if !task.isFile {
			continue
		}
		total++
		switch task.state {
		case TaskSuccess, TaskSkipped, TaskFailed, TaskCanceled:
			done++
		}
		if task.state == TaskFailed {
			failed++
		}
	}
	return
}

func (db *DashboardPanel) totalProgress(tasks []*dashboardTask) (downloaded int64, total int64) {
	for _, task := range tasks {
		if !task.isFile {
			continue
		}
		if task.total > 0 {
			total += task.total
			if task.state == TaskSuccess || task.state == TaskSkipped {
				// 成功、跳过的文件，统计上当做已经下载完成
				downloaded += task.total
			} else {
				if task.downloaded >= task.total {
					downloaded += task.total
				} else {
					downloaded += task.downloaded
				}
			}
		}
	}
	return
}

func (db *DashboardPanel) totalSpeed(tasks []*dashboardTask) int64 {
	if db.globalSpeeds != nil {
		return db.globalSpeeds.GetSpeeds()
	}
	var sum int64
	for _, task := range tasks {
		sum += task.speed
	}
	return sum
}

func (db *DashboardPanel) pickActiveTasks(tasks []*dashboardTask) []*dashboardTask {
	slotCap := db.activeSlots
	if slotCap < 1 {
		slotCap = 1
	}
	active := make([]*dashboardTask, 0, slotCap)
	queued := make([]*dashboardTask, 0, slotCap)
	for _, task := range tasks {
		if !task.isFile {
			continue
		}
		switch task.state {
		case TaskRunning:
			active = append(active, task)
		case TaskQueued:
			queued = append(queued, task)
		}
	}
	active = append(active, queued...)
	if len(active) > slotCap {
		active = active[:slotCap]
	}
	return active
}

func renderTaskLine(task *dashboardTask, index int, width int) string {
	if width < 20 {
		width = 20
	}
	prefix := fmt.Sprintf("%2d. ", index+1)
	name := task.name
	if name == "" {
		name = baseName(task.path)
	}

	// 百分比
	progressPct := 0.0
	percent := "--.-%"
	if task.total > 0 {
		progressPct = float64(task.downloaded) / float64(task.total)
		percent = fmt.Sprintf("%5.1f%%", math.Min(100, progressPct*100)) // 将浮点数格式化为：总宽度 5 个字符，包含 1 位小数，后面加百分号
	} else {
		// 空闲队列左边多一个空格，对齐活跃队列
		percent = " --.-%"
	}

	// 速度
	rate := "-"
	// 剩余时间
	eta := "-"
	switch task.state {
	case TaskQueued:
		rate = "等待中"
	case TaskRunning:
		if task.speed > 0 {
			rate = converter.ConvertFileSize(task.speed, 2) + "/s"
		} else {
			rate = "0 B/s"
		}
		if task.eta > 0 {
			eta = formatDurationShort(task.eta)
		}
	case TaskSuccess:
		rate = "done"
	case TaskFailed:
		rate = "failed"
	case TaskSkipped:
		rate = "skipped"
	case TaskCanceled:
		rate = "canceled"
	}

	barWidth := 20
	rateWidth := 12
	etaWidth := 8
	percentWidth := displayWidth(percent)
	fixed := displayWidth(prefix) + 1 + 2 /*进度条左右[]括号*/ + barWidth + 1 + percentWidth + 1 + rateWidth + 1 + etaWidth
	nameWidth := width - fixed
	if nameWidth < 8 {
		reduce := 8 - nameWidth
		if barWidth > 10 {
			cut := minInt(reduce, barWidth-10)
			barWidth -= cut
			reduce -= cut
		}
		if rateWidth > 8 && reduce > 0 {
			cut := minInt(reduce, rateWidth-8)
			rateWidth -= cut
			reduce -= cut
		}
		if etaWidth > 5 && reduce > 0 {
			cut := minInt(reduce, etaWidth-5)
			etaWidth -= cut
			reduce -= cut
		}
		fixed = displayWidth(prefix) + 1 /*1个空格分隔*/ + 2 /*进度条左右[]括号*/ + barWidth /*进度条长度*/ + 1 /*1个空格分隔*/ + percentWidth + 1 /*1个空格分隔*/ + rateWidth + 1 /*1个空格分隔*/ + etaWidth
		nameWidth = width - fixed
		if nameWidth < 4 {
			nameWidth = 4
		}
	}

	name = padRight(truncateText(name, nameWidth), nameWidth)
	bar := renderBar(progressPct, barWidth)
	rate = padRight(truncateText(rate, rateWidth), rateWidth)
	eta = padRight(truncateText(eta, etaWidth), etaWidth)
	line := fmt.Sprintf("%s%s [%s] %s %s %s", prefix, name, bar, percent, rate, eta)
	return fitLine(line, width)
}

// renderBar 绘制进度条
func renderBar(pct float64, width int) string {
	if width < 5 {
		width = 5
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}
	filled := int(math.Floor(pct * float64(width)))
	// 已经下载的部分使用 => 表示，没有下载的部分使用 空格 表示
	if filled >= 1 {
		return strings.Repeat("=", filled-1) + ">" + strings.Repeat(" ", width-filled)
	}
	return strings.Repeat(" ", width-filled)
}

func formatDurationShort(d time.Duration) string {
	if d <= 0 {
		return "-"
	}
	seconds := int(d.Seconds())
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := seconds / 60
	seconds = seconds % 60
	if minutes < 60 {
		return fmt.Sprintf("%dm%02ds", minutes, seconds)
	}
	hours := minutes / 60
	minutes = minutes % 60
	if hours < 100 {
		return fmt.Sprintf("%dh%02dm", hours, minutes)
	}
	return fmt.Sprintf("%dh", hours)
}

func truncateText(text string, max int) string {
	if max <= 0 || text == "" {
		return ""
	}
	if displayWidth(text) <= max {
		return text
	}
	if max <= 3 {
		return takeWidth(text, max)
	}
	return takeWidth(text, max-3) + "..."
}

func padRight(text string, width int) string {
	if width <= 0 {
		return text
	}
	if displayWidth(text) >= width {
		return text
	}
	return text + strings.Repeat(" ", width-displayWidth(text))
}

func fitLine(text string, width int) string {
	return padRight(truncateText(text, width), width)
}

func displayWidth(text string) int {
	width := 0
	for _, r := range text {
		width += runeWidth(r)
	}
	return width
}

func takeWidth(text string, max int) string {
	if max <= 0 {
		return ""
	}
	used := 0
	for idx, r := range text {
		w := runeWidth(r)
		if used+w > max {
			return text[:idx]
		}
		used += w
	}
	return text
}

func runeWidth(r rune) int {
	if r == 0 {
		return 0
	}
	if r < 0x1100 {
		return 1
	}
	switch {
	case r >= 0x1100 && r <= 0x115f,
		r == 0x2329 || r == 0x232a,
		r >= 0x2e80 && r <= 0xa4cf && r != 0x303f,
		r >= 0xac00 && r <= 0xd7a3,
		r >= 0xf900 && r <= 0xfaff,
		r >= 0xfe10 && r <= 0xfe19,
		r >= 0xfe30 && r <= 0xfe6f,
		r >= 0xff00 && r <= 0xff60,
		r >= 0xffe0 && r <= 0xffe6,
		r >= 0x20000 && r <= 0x2fffd,
		r >= 0x30000 && r <= 0x3fffd:
		return 2
	}
	return 1
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func baseName(path string) string {
	if path == "" {
		return ""
	}
	path = strings.ReplaceAll(path, "\\", "/")
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[idx+1:]
	}
	return path
}
