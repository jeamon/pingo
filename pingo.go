package main

// Pingo is a small & light go-based tool for IP reachability administration tasks with rich user interface.

// Version  : 1.0.0
// Author   : Jerome AMON
// Created  : 19 November 2021

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jroimartin/gocui"
)

const (
	IPLIST  = "ips"
	STATS   = "stats"
	INFOS   = "infos"
	CONFIG  = "config"
	OUTPUTS = "outputs"
	HELP    = "help"

	IPSWIDTH = 22
	HWIDTH   = 46
	HHEIGHT  = 35
)

const helpDetails = `
-------------+------------------------------
    CTRL + A | add multiple ip addresses
-------------+------------------------------
    CTRL + D | delete focused ip address
-------------+------------------------------
    CTRL + E | edit focused ip's configs
-------------+------------------------------
    CTRL + F | search an ip and focus on
-------------+------------------------------
    CTRL + L | load & add ip from files
-------------+------------------------------
    CTRL + Q | close help or stop action 
-------------+------------------------------
    CTRL + P | start pinging focused ip
-------------+------------------------------
    CTRL + R | clear outputs view content
-------------+------------------------------
    CTRL + T | traceroute the focused ip
-------------+------------------------------
    F1 & Esc | display or close help view
-------------+------------------------------
    <Enter>  | start pinging focused ip
-------------+------------------------------
    P or T   | Ping or Trace focused ip
-------------+------------------------------
    Tab Key  | move focus between views
-------------+------------------------------
    ↕ and ↔  | navigate into the IP list
-------------+------------------------------
    CTRL + C | close the full program
-------------+------------------------------

::::::: Crafted with ♥ by Jerome Amon ::::::
`

type config struct {
	start     string
	requests  int
	threshold int
	timeout   int
	size      int
	backup    bool
}

type stat struct {
	min   int
	avg   int
	max   int
	fails int
	match int
	above int
	under int
}

var (
	// global datastore.
	dbs *databases

	// cursor Y line.
	focusedIPChan = make(chan string, 10)

	// IP to ping and to trace.
	ipToPingChan    = make(chan string, 1)
	ipToTraceChan   = make(chan string, 1)
	currentOnPingIP string

	// ping and traceroute output entries.
	outputsDataChan = make(chan string, 10)

	// ping output entries for statistics.
	outputsStatsChan   = make(chan string, 10)
	clearStatsViewChan = make(chan struct{})

	// cleanup outputs view.
	clearOutputsViewChan = make(chan struct{})
	// stop ongoing processing (ping or trace).
	stopProcessingChan = make(chan struct{})

	// custom title of output view.
	outputsTitleChan = make(chan string, 1)

	// control all goroutines.
	exit = make(chan struct{})
	wg   sync.WaitGroup

	LinuxShell = "/bin/sh"
)

// struct of a datastore.
type databases struct {
	ips     map[string]struct{}
	configs map[string]*config
	stats   map[string]*stat
	ipslock *sync.RWMutex
	cfglock *sync.RWMutex
	slock   *sync.RWMutex
}

// newDatabases creates new databases.
func newDatabases() *databases {
	return &databases{
		ips:     map[string]struct{}{},
		configs: make(map[string]*config),
		stats:   make(map[string]*stat),
		ipslock: &sync.RWMutex{},
		cfglock: &sync.RWMutex{},
		slock:   &sync.RWMutex{},
	}
}

// isExists checks if given ip exists.
func (db *databases) isExistsIP(ip string) bool {

	// check if ip is present.
	db.ipslock.RLock()
	if _, ok := db.ips[ip]; ok {
		db.ipslock.RUnlock()
		return true
	}
	db.ipslock.RUnlock()

	return false
}

// addOneMoreIPs take a string of comma-separated IPs and
// initialize their configs & stats then add them.
func (db *databases) addOneMoreIPs(ips string) {
	ipList := strings.Split(ips, ",")
	if len(ipList) == 0 {
		return
	}

	for _, ip := range ipList {
		db.addNewIP(ip)
	}
}

// addNewIP inserts a new ip with its initial configs & stats.
func (db *databases) addNewIP(ip string) {
	ip = strings.TrimSpace(ip)
	if !isValidIP(ip) || db.isExistsIP(ip) {
		return
	}

	db.addIP(ip)
	db.addConfig(ip)
	db.addStats(ip)
}

// addIP inserts a new ip with empty struct as value.
func (db *databases) addIP(ip string) {
	db.ipslock.Lock()
	db.ips[ip] = struct{}{}
	db.ipslock.Unlock()
}

// addConfig inserts a new ip with 0 values as initial configs.
func (db *databases) addConfig(ip string) {
	db.cfglock.Lock()
	db.configs[ip] = &config{start: "n/a"}
	db.cfglock.Unlock()
}

// addStats inserts a new ip with 0 values as initial stats.
func (db *databases) addStats(ip string) {
	db.slock.Lock()
	db.stats[ip] = &stat{}
	db.slock.Unlock()
}

// getJob retrieves a given job data based on its id from jobs store.
func (db *databases) getConfig(ip string) *config {
	var cfg *config
	db.cfglock.RLock()
	cfg = db.configs[ip]
	db.cfglock.RUnlock()
	return cfg
}

// getAction retrieves a given action data based on its id from actions store.
func (db *databases) getStats(ip string) *stat {
	var s *stat
	db.slock.RLock()
	s = db.stats[ip]
	db.slock.RUnlock()
	return s
}

// getAllIPs returns a sorted (by length) list of current IPs.
func (db *databases) getAllIPs() []string {
	dbs.ipslock.RLock()
	ips := make([]string, 0, len(dbs.ips))
	for ip, _ := range dbs.ips {
		ips = append(ips, ip)
	}
	dbs.ipslock.RUnlock()
	sort.Strings(ips)
	sort.SliceStable(ips, func(i, j int) bool {
		return len(ips[i]) < len(ips[j])
	})

	return ips
}

// deleteOneMoreIPs take a string of comma-separated IPs
// and remove them completely from the database.
func (db *databases) deleteOneMoreIPs(ips string) {
	ipList := strings.Split(ips, ",")
	if len(ipList) == 0 {
		return
	}

	for _, ip := range ipList {
		if ip == currentOnPingIP {
			continue
		}
		db.deleteIP(ip)
	}
}

// deleteIP remove completely an ip from datastore.
func (db *databases) deleteIP(ip string) {
	ip = strings.TrimSpace(ip)
	if !isValidIP(ip) || !db.isExistsIP(ip) {
		return
	}

	// remove from ips.
	db.ipslock.Lock()
	delete(db.ips, ip)
	db.ipslock.Unlock()

	// remove from configs.
	db.cfglock.Lock()
	delete(db.configs, ip)
	db.cfglock.Unlock()

	// remove from stats.
	db.slock.Lock()
	delete(db.stats, ip)
	db.slock.Unlock()
}

// isValidIP returns true if ip is valid.
func isValidIP(ip string) bool {
	if net.ParseIP(ip) != nil {
		return true
	}

	return false
}

// formatIPConfig formats a given IP configuration.
func (db *databases) formatIPConfig(ip string) string {
	cfg := db.getConfig(ip)
	return fmt.Sprintf("backup   : %v\ntimeout  : %d\nstarted  : %s\nrequests : %d\npkts size: %d\nthreshold: %d",
		cfg.backup, cfg.timeout, cfg.start, cfg.requests, cfg.size, cfg.threshold)
}

// formatIPStats formats a given IP statistics.
func (db *databases) formatIPStats(ip string) string {
	s := db.getStats(ip)
	return fmt.Sprintf("min  : %d\navg  : %d\nmax  : %d\nfails: %d\nmatch: %d\nabove: %d\nunder: %d\n",
		s.min, s.avg, s.max, s.fails, s.match, s.above, s.under)
}

// loadInitialInfos is called at startup and loads any data piped
// and from all files passed as arguments then fill the databases
// of IP infos with only valid IP addresses.
func (db *databases) loadInitialInfos() {

	// retrieve standard input info.
	fi, _ := os.Stdin.Stat()
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		var entries []string
		// there is data from pipe input, so grab the
		// full content and build a list of entries.
		content, _ := ioutil.ReadAll(os.Stdin)
		entries = strings.Split(string(content), "\n")
		// keep only valid IP addresses.
		for _, e := range entries {
			if isValidIP(strings.TrimSpace(e)) {
				db.addNewIP(strings.TrimSpace(e))
			}
		}
	}

	// parse any files content.
	db.loadInfosFromFiles(os.Args[1:])
}

// loadInfosFromFiles loads data from all files passed as
// input on <CTRL+L> press and fill the databases of IP infos
// with only valid IP addresses.
func (db *databases) loadInfosFromFiles(filenames []string) {

	if len(filenames) == 0 {
		return
	}

	// for each valid file path, grab its full
	// content and build a list of entries.
	var lines []string
	var entries []string
	for _, file := range filenames {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			continue
		}
		// construct the list based on "\n" as sep.
		// then add lines content to entries list.
		lines = strings.Split(string(content), "\n")
		entries = append(entries, lines...)
	}

	if len(entries) == 0 {
		// no data input.
		return
	}

	// keep only valid IP addresses.
	for _, e := range entries {
		if isValidIP(strings.TrimSpace(e)) {
			db.addNewIP(strings.TrimSpace(e))
		}
	}
}

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	// on windows only change terminal title.
	if runtime.GOOS == "windows" {
		exec.Command("cmd", "/c", "title [ PinGo By Jerome Amon ]").Run()
	}

	f, err := os.OpenFile("logs.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Println("failed to create logs file.")
	}
	defer f.Close()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(f)

	// for linux-based platform lets find the current shell binary path
	// if environnement shell is set and not empty we use it as default.
	if runtime.GOOS != "windows" {
		if len(os.Getenv("SHELL")) > 0 {
			LinuxShell = os.Getenv("SHELL")
		}
	}

	// init databases and loads any passed infos.
	dbs = newDatabases()
	dbs.loadInitialInfos()

	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.Highlight = true
	g.SelFgColor = gocui.ColorRed
	g.BgColor = gocui.ColorBlack
	g.FgColor = gocui.ColorWhite
	g.InputEsc = true
	// g.Mouse = true
	g.Cursor = false

	g.SetManagerFunc(layout)

	err = g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit)
	if err != nil {
		log.Println("Could not set key [CtrlC] binding to main view:", err)
		return
	}

	maxX, maxY := g.Size()

	// IPs list view.
	ipsView, err := g.SetView(IPLIST, 0, 0, IPSWIDTH, maxY-19)
	if err != nil && err != gocui.ErrUnknownView {
		log.Println("Failed to create ips list view:", err)
		return
	}
	ipsView.Title = " IP Addresses "
	ipsView.FgColor = gocui.ColorYellow
	ipsView.SelBgColor = gocui.ColorGreen
	ipsView.SelFgColor = gocui.ColorBlack
	ipsView.Highlight = true

	// Outputs view.
	outputsView, err := g.SetView(OUTPUTS, IPSWIDTH+1, 0, maxX-1, maxY-1)
	if err != nil && err != gocui.ErrUnknownView {
		log.Println("Failed to create outputs view:", err)
		return
	}
	outputsView.Title = " Ping Outputs "
	outputsView.FgColor = gocui.ColorYellow
	outputsView.SelBgColor = gocui.ColorGreen
	outputsView.SelFgColor = gocui.ColorBlack
	outputsView.Autoscroll = true
	outputsView.Wrap = false
	outputsView.Highlight = true

	// Current Ping Configs view.
	configView, err := g.SetView(CONFIG, 0, maxY-18, IPSWIDTH, maxY-11)
	if err != nil && err != gocui.ErrUnknownView {
		log.Println("Failed to create config view:", err)
		return
	}
	configView.Title = " Configs "
	configView.FgColor = gocui.ColorYellow
	configView.SelBgColor = gocui.ColorGreen
	configView.SelFgColor = gocui.ColorBlack
	configView.Highlight = false

	// Current Ping Statistics view.
	statsView, err := g.SetView(STATS, 0, maxY-10, IPSWIDTH, maxY-2)
	if err != nil && err != gocui.ErrUnknownView {
		log.Println("Failed to create stats view:", err)
		return
	}
	statsView.Title = " Stats "
	statsView.FgColor = gocui.ColorYellow
	statsView.SelBgColor = gocui.ColorGreen
	statsView.SelFgColor = gocui.ColorBlack
	statsView.Highlight = false
	statsView.Editable = false

	// Infos view.
	infosView, err := g.SetView(INFOS, 0, maxY-2, IPSWIDTH, maxY)
	if err != nil && err != gocui.ErrUnknownView {
		log.Println("Failed to create infos view:", err)
		return
	}
	infosView.FgColor = gocui.ColorRed
	infosView.Highlight = false
	infosView.Editable = false
	infosView.Frame = false
	fmt.Fprint(infosView, " Press F1 For Help ")

	// Apply keybindings to ui.
	if err = keybindings(g); err != nil {
		log.Panicln(err)
	}

	// move the focus on the jobs list box.
	if _, err = g.SetCurrentView(IPLIST); err != nil {
		log.Println("Failed to set focus on ips view:", err)
		return
	}
	// set the cursor & origin to highlight first IP.
	ipsView.SetCursor(0, 0)
	ipsView.SetOrigin(0, 0)

	// display current ips.
	g.Update(updateIPsView)

	wg.Add(1)
	go scheduler()

	wg.Add(1)
	go updateConfigView(g, configView)

	wg.Add(1)
	go updateOutputsView(g, outputsView)

	wg.Add(1)
	go updateStatsView(g, statsView)

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		close(exit)
		log.Println("Exited from the main loop:", err)
	}

	wg.Wait()
}

// updateIPsView loads and displays all ips.
// Formats each IP - 15 witdh and left align.
func updateIPsView(g *gocui.Gui) error {
	v, err := g.View(IPLIST)
	if err != nil {
		log.Println("Failed to update list of ips:", err)
		return err
	}

	v.Clear()

	ips := dbs.getAllIPs()
	for i, ip := range ips {
		fmt.Fprintln(v, fmt.Sprintf("[%02d] %-15s", i, ip))
	}

	return nil
}

// updateConfigView displays focused IP configs.
func updateConfigView(g *gocui.Gui, configView *gocui.View) {
	defer wg.Done()
	var ip string
	for {
		select {
		case <-exit:
			return
		case ip = <-focusedIPChan:
			g.Update(func(g *gocui.Gui) error {
				configView.Clear()
				fmt.Fprint(configView, dbs.formatIPConfig(ip))
				return nil
			})
		}

		time.Sleep(10 * time.Millisecond)
	}
}

// updateOutputsView displays each ping execution output.
// It cleans the outputs view when requested.
func updateOutputsView(g *gocui.Gui, outputsView *gocui.View) {
	defer wg.Done()
	var output string
	for {
		select {
		case output = <-outputsDataChan:
			g.Update(func(g *gocui.Gui) error {
				fmt.Fprint(outputsView, "\n"+output)
				return nil
			})
		case <-clearOutputsViewChan:
			g.Update(func(g *gocui.Gui) error {
				outputsView.Clear()
				outputsView.SetCursor(0, 0)
				outputsView.SetOrigin(0, 0)
				return nil
			})
		case title := <-outputsTitleChan:
			g.Update(func(g *gocui.Gui) error {
				outputsView.Title = title
				return nil
			})
		case <-exit:
			return
		}
		// pause the infinite loop to avoid cpu spike.
		time.Sleep(10 * time.Millisecond)
	}
}

// updateStatsView displays ongoing Ping statistics.
func updateStatsView(g *gocui.Gui, statsView *gocui.View) {
	defer wg.Done()
	var data string
	//var latestStats stats
	for {
		select {
		case data = <-outputsStatsChan:
			g.Update(func(g *gocui.Gui) error {
				if ip, ok := buildStats(data); ok {
					statsView.Clear()
					fmt.Fprint(statsView, dbs.formatIPStats(ip))
				}
				return nil
			})
		case <-clearStatsViewChan:
			//latestStats = &stats{}
			g.Update(func(g *gocui.Gui) error {
				statsView.Clear()
				return nil
			})
		case <-exit:
			return
		}

		time.Sleep(10 * time.Millisecond)
	}
}

// buildStats updates the Ping statistics from a given response data.
// rt == -1 means the output is not a successful reply.
// true means the output states for a ping failure.
// false means to ignore the output (statistics data).
func buildStats(data string) (string, bool) {
	ip, threshold, output := strings.Split(data, "@")[0], strings.Split(data, "@")[1], strings.Split(data, "@")[2]
	stats := dbs.getStats(ip)
	rt, failed := getResponseTime(output)
	if rt == -1 && !failed {
		// ignore output.
		return ip, false
	}
	if rt == -1 && failed {
		// failure response.
		stats.fails += 1
		return ip, true
	}

	// reply response.

	modif := false
	if stats.min == 0 && stats.max == 0 {
		// matches the first output data.
		stats.min, stats.max = rt, rt
		modif = true
	} else {
		// this for following outputs.
		if rt < stats.min {
			stats.min = rt
			modif = true
		} else if stats.max < rt {
			stats.max = rt
			modif = true
		}
	}
	// compute average only if there was a change.
	if modif {
		stats.avg = (stats.min + stats.max) / 2
	}

	thres, _ := strconv.Atoi(threshold)
	if rt == thres {
		stats.match += 1
	} else if rt > thres {
		stats.above += 1
	} else if rt < thres {
		stats.under += 1
	}

	return ip, true
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	// IPs list view.
	_, err := g.SetView(IPLIST, 0, 0, IPSWIDTH, maxY-19)
	if err != nil && err != gocui.ErrUnknownView {
		log.Println("Failed to create ips list view:", err)
		return err
	}

	// Outputs view.
	_, err = g.SetView(OUTPUTS, IPSWIDTH+1, 0, maxX-1, maxY-1)
	if err != nil && err != gocui.ErrUnknownView {
		log.Println("Failed to create outputs view:", err)
		return err
	}

	// Current Ping Configs view.
	_, err = g.SetView(CONFIG, 0, maxY-18, IPSWIDTH, maxY-11)
	if err != nil && err != gocui.ErrUnknownView {
		log.Println("Failed to create config view:", err)
		return err
	}

	// Current Ping Statistics view.
	_, err = g.SetView(STATS, 0, maxY-10, IPSWIDTH, maxY-2)
	if err != nil && err != gocui.ErrUnknownView {
		log.Println("Failed to create stats view:", err)
		return err
	}

	// Infos view.
	_, err = g.SetView(INFOS, 0, maxY-2, IPSWIDTH, maxY)
	if err != nil && err != gocui.ErrUnknownView {
		log.Println("Failed to create infos view:", err)
		return err
	}

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	close(exit)
	return gocui.ErrQuit
}

// keybindings binds multiple keys to views.
func keybindings(g *gocui.Gui) error {

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}

	if err := g.SetKeybinding("", gocui.KeyTab, gocui.ModNone, nextView); err != nil {
		return err
	}

	// use F1 to display help message when the focus is on OUPUTS or IPLIST view.
	if err := g.SetKeybinding(IPLIST, gocui.KeyF1, gocui.ModNone, displayHelpView); err != nil {
		return err
	}

	if err := g.SetKeybinding(OUTPUTS, gocui.KeyF1, gocui.ModNone, displayHelpView); err != nil {
		return err
	}

	// Ctrl+A to create & add one or more new ip addresses (comma-separated input).
	if err := g.SetKeybinding(IPLIST, gocui.KeyCtrlA, gocui.ModNone, addIPInputView); err != nil {
		return err
	}

	if err := g.SetKeybinding(OUTPUTS, gocui.KeyCtrlA, gocui.ModNone, addIPInputView); err != nil {
		return err
	}

	// Ctrl+D to delete one or more existing ip addresses (comma-separated input).
	if err := g.SetKeybinding(IPLIST, gocui.KeyCtrlD, gocui.ModNone, deleteIPInputView); err != nil {
		return err
	}

	if err := g.SetKeybinding(OUTPUTS, gocui.KeyCtrlD, gocui.ModNone, deleteIPInputView); err != nil {
		return err
	}

	// Ctrl+F to find and move cursor on existing ip address.
	if err := g.SetKeybinding(IPLIST, gocui.KeyCtrlF, gocui.ModNone, searchIPInputView); err != nil {
		return err
	}

	if err := g.SetKeybinding(OUTPUTS, gocui.KeyCtrlF, gocui.ModNone, searchIPInputView); err != nil {
		return err
	}

	// Ctrl+L to load new IP infos from a set of files entered into an input box.
	if err := g.SetKeybinding(IPLIST, gocui.KeyCtrlL, gocui.ModNone, loadIPsInputView); err != nil {
		return err
	}

	if err := g.SetKeybinding(OUTPUTS, gocui.KeyCtrlL, gocui.ModNone, loadIPsInputView); err != nil {
		return err
	}

	// Ctrl+R to clear the outputs view content.
	if err := g.SetKeybinding(OUTPUTS, gocui.KeyCtrlR, gocui.ModNone, clearOutputsView); err != nil {
		return err
	}

	// Press <Enter> key or <P> or <Ctrl+P> to add current focused IP to Ping scheduler.
	if err := g.SetKeybinding(IPLIST, gocui.KeyEnter, gocui.ModNone, addPing); err != nil {
		return err
	}

	if err := g.SetKeybinding(IPLIST, gocui.KeyCtrlP, gocui.ModNone, addPing); err != nil {
		return err
	}

	if err := g.SetKeybinding(IPLIST, 'P', gocui.ModNone, addPing); err != nil {
		return err
	}

	// Press <T> key or <Ctrl+T> to add current focused IP to Traceroute scheduler.
	if err := g.SetKeybinding(IPLIST, 'T', gocui.ModNone, addTraceroute); err != nil {
		return err
	}

	if err := g.SetKeybinding(IPLIST, gocui.KeyCtrlT, gocui.ModNone, addTraceroute); err != nil {
		return err
	}

	// arrow keys binding to navigate over the list of items.
	if err := g.SetKeybinding(IPLIST, gocui.KeyArrowUp, gocui.ModNone, ipsMoveCursorUp); err != nil {
		return err
	}

	if err := g.SetKeybinding(IPLIST, gocui.KeyArrowDown, gocui.ModNone, ipsMoveCursorDown); err != nil {
		return err
	}

	if err := g.SetKeybinding(OUTPUTS, gocui.KeyArrowUp, gocui.ModNone, outMoveCursorUp); err != nil {
		return err
	}

	if err := g.SetKeybinding(OUTPUTS, gocui.KeyArrowDown, gocui.ModNone, outMoveCursorDown); err != nil {
		return err
	}

	// stop current ongoing action (if any) - which could be Ping or Traceroute.
	if err := g.SetKeybinding(IPLIST, gocui.KeyCtrlQ, gocui.ModNone, stopCurrentProcessing); err != nil {
		return err
	}

	if err := g.SetKeybinding(OUTPUTS, gocui.KeyCtrlQ, gocui.ModNone, stopCurrentProcessing); err != nil {
		return err
	}

	if err := g.SetKeybinding(CONFIG, gocui.KeyCtrlQ, gocui.ModNone, stopCurrentProcessing); err != nil {
		return err
	}

	if err := g.SetKeybinding(STATS, gocui.KeyCtrlQ, gocui.ModNone, stopCurrentProcessing); err != nil {
		return err
	}

	return nil
}

// stopCurrentProcessing triggered on CTRL+Q send stop flag to channel.
func stopCurrentProcessing(g *gocui.Gui, v *gocui.View) error {
	stopProcessingChan <- struct{}{}
	currentOnPingIP = ""
	return nil
}

// displayHelpView displays help details but trying to center it.
func displayHelpView(g *gocui.Gui, cv *gocui.View) error {

	maxX, maxY := g.Size()

	// construct the input box and position at the center of the screen.
	if helpView, err := g.SetView(HELP, (maxX-HWIDTH)/2, (maxY-HHEIGHT)/2, maxX/2+HWIDTH, (maxY+HHEIGHT)/2); err != nil {
		if err != gocui.ErrUnknownView {
			log.Println("Failed to create help view:", err)
			return err
		}

		helpView.FgColor = gocui.ColorGreen
		helpView.SelBgColor = gocui.ColorBlack
		helpView.SelFgColor = gocui.ColorYellow
		helpView.Editable = false
		helpView.Autoscroll = true
		helpView.Wrap = true
		helpView.Frame = false

		if _, err := g.SetCurrentView(HELP); err != nil {
			log.Println("Failed to set focus on help view:", err)
			return err
		}
		g.Cursor = false

		// bind Ctrl+Q and Escape and F1 keys to close the input box.
		if err := g.SetKeybinding(HELP, gocui.KeyCtrlQ, gocui.ModNone, closeHelpView); err != nil {
			log.Println("Failed to bind keys (CtrlQ) to help view:", err)
			return err
		}

		if err := g.SetKeybinding(HELP, gocui.KeyF1, gocui.ModNone, closeHelpView); err != nil {
			log.Println("Failed to bind keys (F1) to help view:", err)
			return err
		}

		if err := g.SetKeybinding(HELP, gocui.KeyEsc, gocui.ModNone, closeHelpView); err != nil {
			log.Println("Failed to bind keys (Esc) to help view:", err)
			return err
		}

		fmt.Fprintf(helpView, helpDetails)

	}
	return nil
}

// closeHelpView closes help view then move the focus on IP list view.
func closeHelpView(g *gocui.Gui, hv *gocui.View) error {

	hv.Clear()
	g.Cursor = false
	g.DeleteKeybindings(hv.Name())
	if err := g.DeleteView(hv.Name()); err != nil {
		log.Println("Failed to delete help view:", err)
		return err
	}

	return setCurrentDefaultView(g)
}

// clearOutputsView clears outputs view content.
func clearOutputsView(g *gocui.Gui, v *gocui.View) error {
	v.Clear()
	return nil
}

// addIPInputView displays a temporary input box to enter
// a comma-separated list of IP addresses.
func addIPInputView(g *gocui.Gui, cv *gocui.View) error {
	maxX, maxY := g.Size()

	const name = "addIP"

	// construct the input box and position at the center of the screen.
	if inputView, err := g.SetView(name, maxX/2-12, maxY/2, maxX/2+12, maxY/2+2); err != nil {
		if err != gocui.ErrUnknownView {
			log.Println("Failed to display input view: ", err)
			return err
		}

		inputView.Title = " Add IP Addresses "
		inputView.FgColor = gocui.ColorYellow
		inputView.SelBgColor = gocui.ColorBlack
		inputView.SelFgColor = gocui.ColorYellow
		inputView.Editable = true

		if _, err := g.SetCurrentView(name); err != nil {
			log.Println(err)
			return err
		}
		g.Cursor = true
		inputView.Highlight = true
		// bind Enter key to processInput function.
		if err := g.SetKeybinding(name, gocui.KeyEnter, gocui.ModNone, processInput); err != nil {
			log.Println(err)
			return err
		}

		// bind Ctrl+Q and Escape keys to close the input box.
		if err := g.SetKeybinding(name, gocui.KeyCtrlQ, gocui.ModNone, closeInputView); err != nil {
			log.Println(err)
			return err
		}

		if err := g.SetKeybinding(name, gocui.KeyEsc, gocui.ModNone, closeInputView); err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}

// deleteIPInputView displays a temporary input box to delete an IP.
func deleteIPInputView(g *gocui.Gui, cv *gocui.View) error {
	maxX, maxY := g.Size()

	const name = "deleteIP"

	// construct the input box and position at the center of the screen.
	if inputView, err := g.SetView(name, maxX/2-12, maxY/2, maxX/2+12, maxY/2+2); err != nil {
		if err != gocui.ErrUnknownView {
			log.Println("Failed to display input view: ", err)
			return err
		}

		inputView.Title = " Delete IP Addresses "
		inputView.FgColor = gocui.ColorYellow
		inputView.SelBgColor = gocui.ColorBlack
		inputView.SelFgColor = gocui.ColorYellow
		inputView.Editable = true

		if _, err := g.SetCurrentView(name); err != nil {
			log.Println(err)
			return err
		}
		g.Cursor = true
		inputView.Highlight = true
		// bind Enter key to processInput function.
		if err := g.SetKeybinding(name, gocui.KeyEnter, gocui.ModNone, processInput); err != nil {
			log.Println(err)
			return err
		}

		// bind Ctrl+Q and Escape keys to close the input box.
		if err := g.SetKeybinding(name, gocui.KeyCtrlQ, gocui.ModNone, closeInputView); err != nil {
			log.Println(err)
			return err
		}

		if err := g.SetKeybinding(name, gocui.KeyEsc, gocui.ModNone, closeInputView); err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}

// searchIPInputView displays a temporary input box to find an IP.
func searchIPInputView(g *gocui.Gui, cv *gocui.View) error {
	maxX, maxY := g.Size()

	const name = "searchIP"

	// construct the input box and position at the center of the screen.
	if inputView, err := g.SetView(name, maxX/2-12, maxY/2, maxX/2+12, maxY/2+2); err != nil {
		if err != gocui.ErrUnknownView {
			log.Println("Failed to display input view: ", err)
			return err
		}

		inputView.Title = " Search IP Address"
		inputView.FgColor = gocui.ColorYellow
		inputView.SelBgColor = gocui.ColorBlack
		inputView.SelFgColor = gocui.ColorYellow
		inputView.Editable = true

		if _, err := g.SetCurrentView(name); err != nil {
			log.Println(err)
			return err
		}
		g.Cursor = true
		inputView.Highlight = true
		// bind Enter key to processInput function.
		if err := g.SetKeybinding(name, gocui.KeyEnter, gocui.ModNone, searchAndFocusIP); err != nil {
			log.Println(err)
			return err
		}

		// bind Ctrl+Q and Escape keys to close the input box.
		if err := g.SetKeybinding(name, gocui.KeyCtrlQ, gocui.ModNone, closeInputView); err != nil {
			log.Println(err)
			return err
		}

		if err := g.SetKeybinding(name, gocui.KeyEsc, gocui.ModNone, closeInputView); err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}

// loadIPsInputView displays a temporary input box to enter
// a comma-separated list of files to load IP addresses from.
func loadIPsInputView(g *gocui.Gui, cv *gocui.View) error {
	maxX, maxY := g.Size()

	const name = "addFiles"

	// construct the input box and position at the center of the screen.
	if inputView, err := g.SetView(name, maxX/2-25, maxY/2, maxX/2+25, maxY/2+2); err != nil {
		if err != gocui.ErrUnknownView {
			log.Println("Failed to display input view: ", err)
			return err
		}

		inputView.Title = " Enter Filenames (Separated By Comma) "
		inputView.FgColor = gocui.ColorYellow
		inputView.SelBgColor = gocui.ColorBlack
		inputView.SelFgColor = gocui.ColorYellow
		inputView.Editable = true

		if _, err := g.SetCurrentView(name); err != nil {
			log.Println(err)
			return err
		}
		g.Cursor = true
		inputView.Highlight = true
		// bind Enter key to processInput function.
		if err := g.SetKeybinding(name, gocui.KeyEnter, gocui.ModNone, processInput); err != nil {
			log.Println(err)
			return err
		}

		// bind Ctrl+Q and Escape keys to close the input box.
		if err := g.SetKeybinding(name, gocui.KeyCtrlQ, gocui.ModNone, closeInputView); err != nil {
			log.Println(err)
			return err
		}

		if err := g.SetKeybinding(name, gocui.KeyEsc, gocui.ModNone, closeInputView); err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}

// processInput takes the buffer content and process it based on input
// view name. It adds/deletes one or more of IP to/from the database.
func processInput(g *gocui.Gui, iv *gocui.View) error {

	// read buffer from the beginning.
	iv.Rewind()

	// ips list view.
	ov, _ := g.View(IPLIST)

	switch iv.Name() {

	case "addIP":

		if strings.TrimSpace(iv.Buffer()) != "" {
			dbs.addOneMoreIPs(iv.Buffer())
		} else {
			// no data entered, so go back.
			addIPInputView(g, ov)
			return nil
		}

	case "deleteIP":

		if strings.TrimSpace(iv.Buffer()) != "" {
			dbs.deleteOneMoreIPs(iv.Buffer())
		} else {
			deleteIPInputView(g, ov)
			return nil
		}

	case "addFiles":

		if strings.TrimSpace(iv.Buffer()) != "" {
			filenames := strings.Split(strings.TrimSpace(iv.Buffer()), ",")
			// remove any space around each filename.
			for i := 0; i < len(filenames); i++ {
				filenames[i] = strings.TrimSpace(filenames[i])
			}
			dbs.loadInfosFromFiles(filenames)
		} else {
			loadIPsInputView(g, ov)
			return nil
		}
	}

	if err := deleteInputView(g, iv); err != nil {
		log.Println("Failed to delete ips input view: ", err)
		return err
	}

	// set back the focus on ips list view.
	if _, err := g.SetCurrentView(IPLIST); err != nil {
		log.Println("Failed to set back focus on ips list view: ", err)
	}

	// updateIPsView(g)
	g.Update(updateIPsView)

	return nil
}

// searchAndFocusIP locates an IP and move cursor on it.
func searchAndFocusIP(g *gocui.Gui, iv *gocui.View) error {
	// read buffer from the beginning.
	iv.Rewind()

	// ips list view.
	ov, _ := g.View(IPLIST)

	input := strings.TrimSpace(iv.Buffer())
	if input == "" || !isValidIP(input) {
		searchIPInputView(g, ov)
		return nil
	}

	if err := deleteInputView(g, iv); err != nil {
		return err
	}

	// set back the focus on ips list view.
	if _, err := g.SetCurrentView(IPLIST); err != nil {
		log.Println("Failed to set back focus on ips list view: ", err)
	}

	// get all current lines of ips list view.
	pos := -1
	lines := ov.BufferLines()
	for i, line := range lines {
		if strings.Contains(strings.TrimSpace(line), input) {
			pos = i
		}
	}

	if pos != -1 {
		ov.SetCursor(0, pos)
	}

	return nil
}

// deleteInputView deletes a temporary input view.
func deleteInputView(g *gocui.Gui, iv *gocui.View) error {
	// clear and delete input view.
	iv.Clear()
	g.Cursor = false
	g.DeleteKeybindings(iv.Name())
	if err := g.DeleteView(iv.Name()); err != nil {
		log.Println("Failed to delete input view: ", err)
		return err
	}
	return nil
}

// nextView moves the focus to another view.
func nextView(g *gocui.Gui, v *gocui.View) error {

	cv := g.CurrentView()

	if cv == nil {
		if _, err := g.SetCurrentView(IPLIST); err != nil {
			log.Printf("Failed to set focus on default (%v) view: %v", IPLIST, err)
			return err
		}
		return nil
	}

	switch cv.Name() {

	case IPLIST:
		// move the focus on Outputs view.
		if _, err := g.SetCurrentView(OUTPUTS); err != nil {
			log.Println("Failed to set focus on outputs view:", err)
			return err
		}

	case OUTPUTS:
		// move the focus on Configs view.
		if _, err := g.SetCurrentView(CONFIG); err != nil {
			log.Println("Failed to set focus on configs view:", err)
			return err
		}

	case CONFIG:
		// move the focus on Stats view.
		if _, err := g.SetCurrentView(STATS); err != nil {
			log.Println("Failed to set focus on stats view:", err)
			return err
		}

	case STATS:
		// move the focus on IPs view.
		if _, err := g.SetCurrentView(IPLIST); err != nil {
			log.Println("Failed to set focus on ips view:", err)
			return err
		}
	}

	return nil
}

// closeInputView close temporary input view and abort change.
func closeInputView(g *gocui.Gui, iv *gocui.View) error {
	// clear the temporary input view.
	iv.Clear()
	// no input, so disbale cursor.
	g.Cursor = false

	// must delete keybindings before the view, or fatal error.
	g.DeleteKeybindings(iv.Name())
	if err := g.DeleteView(iv.Name()); err != nil {
		log.Println("Failed to delete input view:", err)
		return err
	}

	return setCurrentDefaultView(g)
}

// setCurrentDefaultView moves the focus on default view.
func setCurrentDefaultView(g *gocui.Gui) error {
	// move back the focus on the jobs list box.
	if _, err := g.SetCurrentView(IPLIST); err != nil {
		log.Println("Failed to set focus on default view:", err)
		return err
	}
	return nil
}

// ipsLineBelow returns true if there is an IP at position y+1.
func ipsLineBelow(v *gocui.View) bool {
	_, cy := v.Cursor()
	if l, _ := v.Line(cy + 1); l != "" {
		focusedIPChan <- strings.Fields(strings.TrimSpace(l))[1]
		return true
	}
	return false
}

// ipsMoveCursorDown moves cursor to (currentY + 1) position if there is data there.
func ipsMoveCursorDown(g *gocui.Gui, v *gocui.View) error {
	if v != nil && ipsLineBelow(v) == true {
		// there is data to next line.
		v.MoveCursor(0, 1, false)
	}

	return nil
}

// ipsLineAbove returns true if there is an IP at position y-1.
func ipsLineAbove(v *gocui.View) bool {
	_, cy := v.Cursor()
	if l, _ := v.Line(cy - 1); l != "" {
		focusedIPChan <- strings.Fields(strings.TrimSpace(l))[1]
		return true
	}
	return false
}

// ipsMoveCursorUp moves cursor to (currentY - 1) position if there is data there.
func ipsMoveCursorUp(g *gocui.Gui, v *gocui.View) error {
	if v != nil && ipsLineAbove(v) == true {
		// there is data upper.
		v.MoveCursor(0, -1, false)
	}

	return nil
}

// lineBelow returns true if there is a non-empty string in cursor position y+1.
func lineBelow(v *gocui.View) bool {
	_, cy := v.Cursor()
	if l, _ := v.Line(cy + 1); l != "" {
		return true
	}
	return false
}

// outMoveCursorDown moves cursor to (currentY + 1) position if there is data there.
func outMoveCursorDown(g *gocui.Gui, v *gocui.View) error {
	if v != nil && lineBelow(v) == true {
		// there is data to next line.
		v.MoveCursor(0, 1, false)
	}

	return nil
}

// lineAbove returns true if there is a non-empty string in cursor position y-1.
func lineAbove(v *gocui.View) bool {
	_, cy := v.Cursor()
	if l, _ := v.Line(cy - 1); l != "" {
		return true
	}
	return false
}

// outMoveCursorUp moves cursor to (currentY - 1) position if there is data there.
func outMoveCursorUp(g *gocui.Gui, v *gocui.View) error {
	if v != nil && lineAbove(v) == true {
		// there is data upper.
		v.MoveCursor(0, -1, false)
	}

	return nil
}

// addPing is triggered when Enter or CTRL+P or <P> key is pressed
// inside IPLIST view. It extracts the exact IP address and add it
// to the channel <ipToPingChan> for ping scheduler.
func addPing(g *gocui.Gui, ipv *gocui.View) error {
	_, cy := ipv.Cursor()
	l, err := ipv.Line(cy)
	if err != nil {
		log.Println("Failed to read current focused ip value:", err)
		return nil
	}
	if len(l) == 0 {
		return nil
	}
	ip := strings.Fields(strings.TrimSpace(l))[1]
	outputsTitleChan <- fmt.Sprintf(" Ping [%s] Outputs ", ip)
	ipToPingChan <- ip
	currentOnPingIP = ip
	focusedIPChan <- ip
	return nil
}

// addTraceroute is triggered when CTRL+T or <T> key is pressed inside
// IPLIST view. It extracts the exact IP address and add it to the
// channel <ipToTraceChan> for traceroute scheduler.
func addTraceroute(g *gocui.Gui, ipv *gocui.View) error {
	_, cy := ipv.Cursor()
	l, err := ipv.Line(cy)
	if err != nil {
		log.Println("Failed to read current focused ip value:", err)
		return nil
	}
	if len(l) == 0 {
		return nil
	}
	ip := strings.Fields(strings.TrimSpace(l))[1]
	outputsTitleChan <- fmt.Sprintf(" Traceroute [%s] Outputs ", ip)
	ipToTraceChan <- ip
	// reset since no ping.
	currentOnPingIP = ""
	return nil
}

// scheduler watches the ping and traceroute jobs channels and spin up
// a separate ping or traceroute executor. It can clear the outputs view
// or just cancel any ongoing processing.
func scheduler() {
	defer wg.Done()
	ctx, cancel := context.WithCancel(context.Background())
	for {
		select {
		case ip := <-ipToPingChan:
			cancel()
			clearOutputsViewChan <- struct{}{}
			clearStatsViewChan <- struct{}{}
			ctx, cancel = context.WithCancel(context.Background())
			go executePing(ip, ctx)
		case ip := <-ipToTraceChan:
			cancel()
			clearOutputsViewChan <- struct{}{}
			clearStatsViewChan <- struct{}{}
			ctx, cancel = context.WithCancel(context.Background())
			go executeTraceroute(ip, ctx)
		case <-stopProcessingChan:
			cancel()
		case <-exit:
			cancel()
			return
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// getCurrentTime returns the current datetime in custom format.
func getCurrentTime() string {
	t := time.Now()
	return fmt.Sprintf("%02d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
}

// buildPingCommand constructs full command to run. The ping should
// run indefinitely by default unless a requests is defined.
func buildPingCommand(ip string, ctx context.Context) (string, *exec.Cmd) {
	cfg := dbs.getConfig(ip)
	cfg.start = getCurrentTime()
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		syntax := fmt.Sprintf("ping %s", ip)

		if cfg.requests > 0 {
			syntax = syntax + fmt.Sprintf(" -n %d", cfg.requests)
		} else {
			syntax = syntax + " -t"
		}

		if cfg.timeout > 0 {
			syntax = syntax + fmt.Sprintf(" -w %d", cfg.timeout)
		}

		if cfg.size > 0 {
			syntax = syntax + fmt.Sprintf(" -l %d", cfg.size)
		}

		cmd = exec.CommandContext(ctx, "cmd", "/C", syntax)
	} else {
		syntax := fmt.Sprintf("ping %s", ip)

		if cfg.requests > 0 {
			syntax = syntax + fmt.Sprintf(" -c %d", cfg.requests)
		}

		if cfg.timeout > 0 {
			syntax = syntax + fmt.Sprintf(" -W %d", cfg.timeout)
		}

		if cfg.size > 0 {
			syntax = syntax + fmt.Sprintf(" -s %d", cfg.size)
		}

		cmd = exec.CommandContext(ctx, LinuxShell, "-c", syntax)
	}

	return strconv.Itoa(cfg.threshold), cmd
}

// executePing runs the full ping command.
func executePing(ip string, ctx context.Context) {

	threshold, cmd := buildPingCommand(ip, ctx)
	// combined outputs.
	cmd.Stderr = cmd.Stdout
	outpipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Println("Failed to get ping process pipe:", err)
		return
	}
	// async start.
	err = cmd.Start()
	if err != nil {
		log.Println("Failed to start ping:", err)
		return
	}

	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	// read each line from the pipe content including
	// the newline char and stream it to data channel.
	go func(ip, threshold string) {
		var data string
		var err error
		reader := bufio.NewReader(outpipe)
		for {
			data, err = reader.ReadString('\n')
			if err != nil {
				return
			}
			if strings.TrimSpace(data) == "" {
				continue
			}
			outputsStatsChan <- ip + "@" + threshold + "@" + strings.TrimSpace(data)
			outputsDataChan <- strings.TrimSpace(data)
		}
	}(ip, threshold)

	select {
	case <-ctx.Done():
		break
	case <-done:
		break
	}

	return
}

// buildTracerouteCommand constructs full command to run.
func buildTracerouteCommand(ip string, ctx context.Context) *exec.Cmd {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", fmt.Sprintf("tracert %s", ip))
	} else {
		cmd = exec.CommandContext(ctx, LinuxShell, "-c", fmt.Sprintf("traceroute %s", ip))
	}

	return cmd
}

// executeTraceroute runs the traceroute command.
func executeTraceroute(ip string, ctx context.Context) {

	cmd := buildTracerouteCommand(ip, ctx)
	cmd.Stderr = cmd.Stdout
	outpipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Println("Failed to get traceroute process pipe:", err)
		return
	}
	// async start.
	err = cmd.Start()
	if err != nil {
		log.Println("Failed to start traceroute:", err)
		return
	}

	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	// read each line from the pipe content including
	// the newline char and stream it to data channel.
	go func() {
		var data string
		var err error
		reader := bufio.NewReader(outpipe)
		for {
			data, err = reader.ReadString('\n')
			if err != nil {
				return
			}
			outputsDataChan <- strings.TrimSpace(data)
		}
	}()

	select {
	case <-ctx.Done():
		return
	case <-done:
		return
	}

	return
}
