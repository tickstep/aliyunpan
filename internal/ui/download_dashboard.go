// Copyright (c) 2026.
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

	"github.com/tickstep/library-go/converter"
	"github.com/tickstep/library-go/requester/rio/speeds"
)

type TaskState int

const (
	TaskQueued TaskState = iota
	TaskRunning
	TaskSuccess
	TaskFailed
	TaskSkipped
	TaskCanceled
)

type DownloadDashboardOptions struct {
	Title       string
	MaxHistory  int
	ActiveSlots int
	Output      io.Writer
}

type DownloadDashboard struct {
	mu           sync.Mutex
	tasks        map[string]*downloadTask
	order        []string
	history      []string
	maxHistory   int
	parallel     int
	activeSlots  int
	globalSpeeds *speeds.Speeds
	title        string
	startTime    time.Time
	renderEvery  time.Duration
	stopCh       chan struct{}
	doneCh       chan struct{}
	startOnce    sync.Once
	closeOnce    sync.Once
	out          io.Writer
}

type downloadTask struct {
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
	hasType    bool
}

func NewDownloadDashboard(parallel int, globalSpeeds *speeds.Speeds, opts *DownloadDashboardOptions) *DownloadDashboard {
	if parallel < 1 {
		parallel = 1
	}
	db := &DownloadDashboard{
		tasks:        make(map[string]*downloadTask),
		order:        make([]string, 0, 128),
		history:      make([]string, 0, 64),
		maxHistory:   8,
		parallel:     parallel,
		activeSlots:  5,
		globalSpeeds: globalSpeeds,
		title:        "AliyunPan Download",
		renderEvery:  500 * time.Millisecond,
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
		out:          os.Stdout,
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

func (db *DownloadDashboard) Start() {
	if db == nil {
		return
	}
	db.startOnce.Do(func() {
		db.startTime = time.Now()
		enableVirtualTerminalProcessing()
		fmt.Fprint(db.out, "\x1b[?25l")
		db.render()
		ticker := time.NewTicker(db.renderEvery)
		go func() {
			defer close(db.doneCh)
			defer ticker.Stop()
			for {
				select {
				case <-db.stopCh:
					db.render()
					return
				case <-ticker.C:
					db.render()
				}
			}
		}()
	})
}

func (db *DownloadDashboard) Close() {
	if db == nil {
		return
	}
	db.closeOnce.Do(func() {
		close(db.stopCh)
		<-db.doneCh
		fmt.Fprint(db.out, "\x1b[?25h")
	})
}

func (db *DownloadDashboard) RegisterTask(id, path string, total int64, isFile bool) {
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
	task.hasType = true
	task.isFile = isFile
}

func (db *DownloadDashboard) UpdateTaskInfo(id, name string, total int64, isFile bool) {
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
	task.hasType = true
	task.isFile = isFile
}

func (db *DownloadDashboard) UpdateTaskProgress(id string, downloaded, total, speed int64, eta time.Duration) {
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

func (db *DownloadDashboard) MarkTaskState(id string, state TaskState, message string) {
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

func (db *DownloadDashboard) Logf(format string, a ...interface{}) {
	if db == nil {
		return
	}
	msg := fmt.Sprintf(format, a...)
	msg = strings.TrimRight(msg, "\r\n")
	if msg == "" {
		return
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	db.appendHistoryLocked(msg)
}

func (db *DownloadDashboard) getOrCreateTaskLocked(id string) *downloadTask {
	if task, ok := db.tasks[id]; ok {
		return task
	}
	task := &downloadTask{
		id:    id,
		state: TaskQueued,
	}
	db.tasks[id] = task
	db.order = append(db.order, id)
	return task
}

func (db *DownloadDashboard) appendHistoryLocked(message string) {
	if message == "" {
		return
	}
	timestamp := time.Now().Format("15:04:05")
	db.history = append(db.history, fmt.Sprintf("[%s] %s", timestamp, message))
	if len(db.history) > db.maxHistory*3 {
		db.history = db.history[len(db.history)-db.maxHistory*2:]
	}
}

func (db *DownloadDashboard) defaultHistoryMessage(task *downloadTask, message string) string {
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

func (db *DownloadDashboard) render() {
	if db == nil {
		return
	}
	width, height := terminalSize()
	if width < 60 {
		width = 60
	}
	innerWidth := width - 2
	snapshot := db.snapshot()

	headerLines := db.headerLines(snapshot, innerWidth)
	activeLines := db.activeLines(snapshot, innerWidth)
	activeTitle := db.activeTitle(snapshot)

	headerHeight := len(headerLines) + 2
	activeHeight := len(activeLines) + 4
	historyBase := 4
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

	builder := &strings.Builder{}
	builder.WriteString("\x1b[H\x1b[2J")
	builder.WriteString(renderBox(headerLines, width))
	builder.WriteString("\n")
	builder.WriteString(renderTitledBox(activeTitle, activeLines, width))
	builder.WriteString("\n")
	builder.WriteString(renderTitledBox("History", historyBody, width))

	fmt.Fprint(db.out, builder.String())
}

type dashboardSnapshot struct {
	tasks   []*downloadTask
	history []string
}

func (db *DownloadDashboard) snapshot() dashboardSnapshot {
	db.mu.Lock()
	defer db.mu.Unlock()
	tasks := make([]*downloadTask, 0, len(db.order))
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
		builder.WriteString("\u2502")
		builder.WriteString(fitLine(line, innerWidth))
		builder.WriteString("\u2502")
	}
	builder.WriteString("\n")
	builder.WriteString("\u2514")
	builder.WriteString(strings.Repeat("\u2500", innerWidth))
	builder.WriteString("\u2518")
	return builder.String()
}

func renderTitledBox(title string, body []string, width int) string {
	if width < 4 {
		width = 4
	}
	innerWidth := width - 2
	builder := &strings.Builder{}
	builder.WriteString("\u250c")
	builder.WriteString(strings.Repeat("\u2500", innerWidth))
	builder.WriteString("\u2510\n")
	builder.WriteString("\u2502")
	builder.WriteString(fitLine(title, innerWidth))
	builder.WriteString("\u2502\n")
	builder.WriteString("\u251c")
	builder.WriteString(strings.Repeat("\u2500", innerWidth))
	builder.WriteString("\u2524")
	for _, line := range body {
		builder.WriteString("\n")
		builder.WriteString("\u2502")
		builder.WriteString(fitLine(line, innerWidth))
		builder.WriteString("\u2502")
	}
	builder.WriteString("\n")
	builder.WriteString("\u2514")
	builder.WriteString(strings.Repeat("\u2500", innerWidth))
	builder.WriteString("\u2518")
	return builder.String()
}

func (db *DownloadDashboard) headerLines(snapshot dashboardSnapshot, innerWidth int) []string {
	totalFiles, doneFiles, failedFiles := db.fileCounts(snapshot.tasks)
	downloaded, total := db.totalProgress(snapshot.tasks)
	speed := db.totalSpeed(snapshot.tasks)
	elapsed := time.Since(db.startTime)
	left := "-"
	if speed > 0 && total > downloaded {
		left = formatDurationShort(time.Duration((total-downloaded)/speed) * time.Second)
	}

	status := "Running"
	if totalFiles > 0 && doneFiles >= totalFiles {
		if failedFiles > 0 {
			status = "Completed (errors)"
		} else {
			status = "Completed"
		}
	}

	headerLine := fmt.Sprintf("%s [%s]", db.title, status)
	line1 := fitLine(headerLine, innerWidth)

	speedStr := converter.ConvertFileSize(speed, 2) + "/s"
	line2 := fmt.Sprintf("Total Speed: %s | Files: %d/%d | Failed: %d | Elapsed: %s | Left: %s",
		speedStr, doneFiles, totalFiles, failedFiles, formatDurationShort(elapsed), left)
	line2 = fitLine(line2, innerWidth)

	progressPct := 0.0
	if total > 0 {
		progressPct = float64(downloaded) / float64(total)
	}
	progress := fmt.Sprintf("%s/%s", converter.ConvertFileSize(downloaded, 2), converter.ConvertFileSize(total, 2))
	label := "Total Progress: "
	suffix := fmt.Sprintf(" %5.1f%% (%s)", progressPct*100, progress)
	barWidth := innerWidth - displayWidth(label) - displayWidth(suffix) - 2
	if barWidth < 10 {
		barWidth = 10
	}
	bar := renderBar(progressPct, barWidth)
	line3 := fmt.Sprintf("%s[%s]%s", label, bar, suffix)
	line3 = fitLine(line3, innerWidth)

	return []string{line1, line2, line3}
}

func (db *DownloadDashboard) activeTitle(snapshot dashboardSnapshot) string {
	running := 0
	queued := 0
	for _, task := range snapshot.tasks {
		if task.hasType && !task.isFile {
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

func (db *DownloadDashboard) activeLines(snapshot dashboardSnapshot, innerWidth int) []string {
	active := db.pickActiveTasks(snapshot.tasks)
	lines := make([]string, 0, db.activeSlots)
	for i := 0; i < db.activeSlots; i++ {
		if i < len(active) {
			lines = append(lines, renderTaskLine(active[i], i, innerWidth))
			continue
		}
		placeholder := &downloadTask{name: "waiting...", state: TaskQueued}
		lines = append(lines, renderTaskLine(placeholder, i, innerWidth))
	}
	return lines
}

func (db *DownloadDashboard) historyLines(snapshot dashboardSnapshot, innerWidth int, lines int) []string {
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
	for len(result) < lines {
		result = append(result, "")
	}
	return result[:lines]
}

func (db *DownloadDashboard) fileCounts(tasks []*downloadTask) (total int, done int, failed int) {
	for _, task := range tasks {
		if task.hasType && !task.isFile {
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

func (db *DownloadDashboard) totalProgress(tasks []*downloadTask) (downloaded int64, total int64) {
	for _, task := range tasks {
		if task.hasType && !task.isFile {
			continue
		}
		if task.total > 0 {
			total += task.total
			if task.downloaded > task.total {
				downloaded += task.total
			} else {
				downloaded += task.downloaded
			}
		}
	}
	return
}

func (db *DownloadDashboard) totalSpeed(tasks []*downloadTask) int64 {
	if db.globalSpeeds != nil {
		return db.globalSpeeds.GetSpeeds()
	}
	var sum int64
	for _, task := range tasks {
		sum += task.speed
	}
	return sum
}

func (db *DownloadDashboard) pickActiveTasks(tasks []*downloadTask) []*downloadTask {
	slotCap := db.activeSlots
	if slotCap < 1 {
		slotCap = 1
	}
	active := make([]*downloadTask, 0, slotCap)
	queued := make([]*downloadTask, 0, slotCap)
	for _, task := range tasks {
		if task.hasType && !task.isFile {
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

func renderTaskLine(task *downloadTask, index int, width int) string {
	if width < 20 {
		width = 20
	}
	prefix := fmt.Sprintf("%2d. ", index+1)
	name := task.name
	if name == "" {
		name = baseName(task.path)
	}

	progressPct := 0.0
	percent := "--.-%"
	if task.total > 0 {
		progressPct = float64(task.downloaded) / float64(task.total)
		percent = fmt.Sprintf("%5.1f%%", math.Min(100, progressPct*100))
	}

	rate := "-"
	eta := "-"
	switch task.state {
	case TaskQueued:
		rate = "waiting"
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
	fixed := displayWidth(prefix) + 1 + 2 + barWidth + 1 + percentWidth + 1 + rateWidth + 1 + etaWidth
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
		fixed = displayWidth(prefix) + 1 + 2 + barWidth + 1 + percentWidth + 1 + rateWidth + 1 + etaWidth
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
	return strings.Repeat("\u2588", filled) + strings.Repeat("\u2591", width-filled)
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
