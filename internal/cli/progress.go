package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

const (
	progressTitleWidth = 30
	progressBarWidth   = 30
	progressSizeWidth  = 14
)

type progressManager struct {
	writer     io.Writer
	mu         sync.Mutex
	outputMu   sync.Mutex
	slots      []progressSlot
	doneBytes  int64
	doneTotal  int64
	doneItems  int
	totalItems int
	started    time.Time
	lastLines  int
	stopCh     chan struct{}
	stopped    chan struct{}
}

type progressSlot struct {
	title      string
	current    int64
	total      int64
	startBytes int64
	started    time.Time
	active     bool
	status     string
}

type progressTracker struct {
	pm   *progressManager
	slot int
}

func newProgressManager(cfg *Config, concurrency int, totalItems int) *progressManager {
	if cfg == nil || cfg.JSON || cfg.Plain || cfg.Quiet {
		return nil
	}
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		return nil
	}
	if concurrency < 1 {
		return nil
	}
	pm := &progressManager{
		writer:     os.Stderr,
		slots:      make([]progressSlot, concurrency),
		started:    time.Now(),
		totalItems: totalItems,
		stopCh:     make(chan struct{}),
		stopped:    make(chan struct{}),
	}
	go pm.run()
	return pm
}

func (pm *progressManager) Stop() {
	if pm == nil {
		return
	}
	close(pm.stopCh)
	<-pm.stopped
	pm.clearLines()
}

func (pm *progressManager) Start(slot int, title string) *progressTracker {
	if pm == nil || slot < 0 || slot >= len(pm.slots) {
		return nil
	}
	pm.mu.Lock()
	pm.slots[slot] = progressSlot{
		title:  title,
		active: true,
	}
	pm.mu.Unlock()
	return &progressTracker{pm: pm, slot: slot}
}

func (pt *progressTracker) Reset(title string) {
	if pt == nil || pt.pm == nil {
		return
	}
	pt.pm.mu.Lock()
	pt.pm.slots[pt.slot] = progressSlot{
		title:      title,
		active:     true,
		started:    time.Now(),
		startBytes: 0,
	}
	pt.pm.mu.Unlock()
}

func (pt *progressTracker) SetTotal(total int64) {
	if pt == nil || pt.pm == nil {
		return
	}
	pt.pm.mu.Lock()
	pt.pm.slots[pt.slot].total = total
	pt.pm.mu.Unlock()
}

func (pt *progressTracker) SetCurrent(current int64) {
	if pt == nil || pt.pm == nil {
		return
	}
	pt.pm.mu.Lock()
	slot := pt.pm.slots[pt.slot]
	slot.current = current
	slot.startBytes = current
	pt.pm.slots[pt.slot] = slot
	pt.pm.mu.Unlock()
}

func (pt *progressTracker) Add(delta int64) {
	if pt == nil || pt.pm == nil || delta <= 0 {
		return
	}
	pt.pm.mu.Lock()
	pt.pm.slots[pt.slot].current += delta
	pt.pm.mu.Unlock()
}

func (pt *progressTracker) SetStatus(status string) {
	if pt == nil || pt.pm == nil {
		return
	}
	pt.pm.mu.Lock()
	pt.pm.slots[pt.slot].status = strings.TrimSpace(status)
	pt.pm.mu.Unlock()
}

func (pt *progressTracker) Finish() {
	if pt == nil || pt.pm == nil {
		return
	}
	pt.pm.finishSlot(pt.slot)
}

func (pm *progressManager) finishSlot(slot int) {
	if pm == nil {
		return
	}
	pm.mu.Lock()
	if slot < 0 || slot >= len(pm.slots) {
		pm.mu.Unlock()
		return
	}
	entry := pm.slots[slot]
	if entry.active {
		pm.doneBytes += entry.current
		if entry.total > 0 {
			pm.doneTotal += entry.total
		}
		pm.doneItems++
	}
	if entry.status != "" {
		entry.current = 0
		entry.total = 0
		entry.startBytes = 0
		entry.started = time.Time{}
		entry.active = true
		pm.slots[slot] = entry
	} else {
		pm.slots[slot] = progressSlot{}
	}
	pm.mu.Unlock()
}

func (pm *progressManager) MarkDone() {
	if pm == nil {
		return
	}
	pm.mu.Lock()
	pm.doneItems++
	pm.mu.Unlock()
}

func (pm *progressManager) run() {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-pm.stopCh:
			close(pm.stopped)
			return
		case <-ticker.C:
			pm.render()
		}
	}
}

func (pm *progressManager) render() {
	lines := pm.snapshotLines()
	pm.writeLines(lines)
}

func (pm *progressManager) snapshotLines() []string {
	pm.mu.Lock()
	slots := make([]progressSlot, len(pm.slots))
	copy(slots, pm.slots)
	doneBytes := pm.doneBytes
	doneTotal := pm.doneTotal
	doneItems := pm.doneItems
	totalItems := pm.totalItems
	started := pm.started
	pm.mu.Unlock()

	var lines []string
	now := time.Now()
	var overallCurrent int64
	var overallTotal int64
	var overallSpeed float64
	progressItems := float64(doneItems)
	for _, slot := range slots {
		if !slot.active {
			continue
		}
		overallCurrent += slot.current
		if slot.total > 0 {
			overallTotal += slot.total
			progressItems += float64(slot.current) / float64(slot.total)
		}
		overallSpeed += slotSpeed(slot, now)
	}
	overallCurrent += doneBytes
	overallTotal += doneTotal

	lines = append(lines, formatOverallLine("Overall", progressItems, doneItems, totalItems, overallSpeed, now, started))
	for _, slot := range slots {
		if !slot.active {
			continue
		}
		lines = append(lines, formatProgressLine(slot.title, slot.current, slot.total, slotSpeed(slot, now), now, slot.started, slot.status))
	}
	return lines
}

func slotSpeed(slot progressSlot, now time.Time) float64 {
	if slot.started.IsZero() {
		return 0
	}
	elapsed := now.Sub(slot.started).Seconds()
	if elapsed <= 0 {
		return 0
	}
	downloaded := float64(slot.current - slot.startBytes)
	if downloaded < 0 {
		downloaded = 0
	}
	return downloaded / elapsed
}

func (pm *progressManager) writeLines(lines []string) {
	if pm == nil {
		return
	}
	pm.outputMu.Lock()
	defer pm.outputMu.Unlock()
	pm.writeLinesLocked(lines)
}

func (pm *progressManager) writeLinesLocked(lines []string) {
	if pm == nil {
		return
	}
	lineCount := len(lines)
	if pm.lastLines > lineCount {
		for i := lineCount; i < pm.lastLines; i++ {
			lines = append(lines, "")
		}
		lineCount = len(lines)
	}
	pm.lastLines = lineCount

	var b strings.Builder
	for i, line := range lines {
		b.WriteString("\r\033[2K")
		b.WriteString(line)
		if i < len(lines)-1 {
			b.WriteString("\n")
		}
	}
	if len(lines) > 1 {
		b.WriteString(fmt.Sprintf("\033[%dA", len(lines)-1))
	}
	_, _ = io.WriteString(pm.writer, b.String())
}

func (pm *progressManager) clearLines() {
	if pm == nil || pm.lastLines == 0 {
		return
	}
	pm.outputMu.Lock()
	defer pm.outputMu.Unlock()
	pm.clearLinesLocked(true)
}

func (pm *progressManager) clearLinesLocked(withNewline bool) {
	if pm == nil || pm.lastLines == 0 {
		return
	}
	lines := make([]string, pm.lastLines)
	pm.lastLines = 0
	pm.writeLinesLocked(lines)
	if withNewline {
		_, _ = io.WriteString(pm.writer, "\n")
	}
}

func (pm *progressManager) PrintLine(line string) {
	if pm == nil {
		return
	}
	lines := pm.snapshotLines()
	pm.outputMu.Lock()
	defer pm.outputMu.Unlock()
	pm.clearLinesLocked(false)
	if !strings.HasSuffix(line, "\n") {
		line += "\n"
	}
	_, _ = io.WriteString(pm.writer, line)
	pm.writeLinesLocked(lines)
}

func formatProgressLine(title string, current, total int64, speed float64, now time.Time, started time.Time, status string) string {
	title = formatTitle(title, progressTitleWidth)
	percent := 0.0
	if total > 0 {
		percent = (float64(current) / float64(total)) * 100
		if percent > 100 {
			percent = 100
		}
	}
	bar := formatBar(percent, progressBarWidth)
	speedStr := formatSpeed(speed)
	sizeStr, eta := formatSizeAndETA(current, total, speed)
	tail := "ETA " + eta
	if status != "" {
		tail = status
	}
	return fmt.Sprintf("%s %5.1f%% [%s] %s %-*s %s", title, percent, bar, speedStr, progressSizeWidth, sizeStr, tail)
}

func formatOverallLine(title string, progress float64, doneItems, totalItems int, speed float64, now time.Time, started time.Time) string {
	title = formatTitle(title, progressTitleWidth)
	percent := 0.0
	if totalItems > 0 {
		percent = (progress / float64(totalItems)) * 100
		if percent > 100 {
			percent = 100
		}
	}
	bar := formatBar(percent, progressBarWidth)
	speedStr := formatSpeed(speed)
	sizeStr := fmt.Sprintf("%d/%d", doneItems, totalItems)
	eta := "--:--:--"
	if totalItems > 0 {
		elapsed := now.Sub(started).Seconds()
		if elapsed > 0 {
			rate := progress / elapsed
			if rate > 0 {
				remaining := float64(totalItems) - progress
				if remaining < 0 {
					remaining = 0
				}
				eta = formatETA(remaining / rate)
			}
		}
	}
	return fmt.Sprintf("%s %5.1f%% [%s] %s %-*s ETA %s", title, percent, bar, speedStr, progressSizeWidth, sizeStr, eta)
}

func formatTitle(title string, width int) string {
	runes := []rune(title)
	if len(runes) <= width {
		return padRight(title, width)
	}
	if width <= 1 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + "…"
}

func formatBar(percent float64, width int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	filled := int((percent / 100) * float64(width))
	if filled > width {
		filled = width
	}
	return strings.Repeat("#", filled) + strings.Repeat(".", width-filled)
}

func formatSpeed(bytesPerSecond float64) string {
	mb := bytesPerSecond / (1024 * 1024)
	if mb < 0 {
		mb = 0
	}
	if mb > 9999.9 {
		mb = 9999.9
	}
	return padRight(fmt.Sprintf("%.1f MB/s", mb), 11)
}

func formatSizeAndETA(current, total int64, speed float64) (string, string) {
	if total <= 0 {
		return "0.0/0.0 MB", "--:--:--"
	}
	unit := "MB"
	divider := float64(1024 * 1024)
	if total >= 1024*1024*1024 {
		unit = "GB"
		divider = float64(1024 * 1024 * 1024)
	}
	curVal := float64(current) / divider
	totalVal := float64(total) / divider
	size := fmt.Sprintf("%.1f/%.1f %s", curVal, totalVal, unit)
	eta := "--:--:--"
	if speed > 0 && current <= total {
		remaining := float64(total-current) / speed
		eta = formatETA(remaining)
	}
	return size, eta
}

func formatETA(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	secs := int64(seconds + 0.5)
	h := secs / 3600
	m := (secs % 3600) / 60
	s := secs % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func padRight(value string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-len(runes))
}
