package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
)

var ids []string = []string{ "I", "J", "F", "G", "H", "K" }

var eNid string
var sections []*cdp.Node = []*cdp.Node{}
var previousSections []*cdp.Node = []*cdp.Node{}
var sameSections int = 1

var soundFile *os.File
var streamer beep.StreamSeekCloser
var format beep.Format

func initSound() func() {
    var err error
    soundFile, err = os.Open("./twitter.mp3")
    if err != nil { log.Fatal(err) }
    streamer, format, err = mp3.Decode(soundFile)
    if err != nil { log.Fatal(err) }
    speaker.Init(format.SampleRate, format.SampleRate.N(time.Second / 10))
    return func() { streamer.Close() }
}

func playSound(ctx context.Context) error {
    speaker.Play(streamer)
    return nil
}

func compareSectionArrays(a []*cdp.Node, b []*cdp.Node) bool {
    if len(a) != len(b) { return false }
    for i := range a { // TODO: ?? hacer mejor este for
        if a[i].AttributeValue("id") != b[i].AttributeValue("id") {
            return false
        }
    }
    return true
}

func printSections(ctx context.Context) error {
    same := compareSectionArrays(previousSections, sections)
    if !same {
        sameSections = 1
        fmt.Printf("\n")
    }
    fmt.Printf("\rsecciones: [")
    for i, node := range sections {
        fmt.Printf("%v", node.AttributeValue("id"))
        if i != len(sections) - 1 { fmt.Printf(", ") }
    }
    fmt.Printf("] x %v", sameSections)
    if same { sameSections++ }
    previousSections = sections
    return nil
}

func contains(s []string, str string) bool {
    for _, v := range s {
        if v == str {
            return true
        }
    }
    return false
}

func getArg(arg string) *string {
    for i, v := range os.Args {
        if v == arg && len(os.Args) >= i + 2 {
            return &os.Args[i + 1]
        }
    }
    return nil
}

func parseArgs() (username *string, password *string) {
    if username = getArg("-u"); username == nil { username = getArg("--username") }
    if username == nil { log.Fatal("Username missing") }
    if password = getArg("-p"); password == nil { password = getArg("--password") }
    if password == nil { log.Fatal("Password missing") }
    return username, password
}

func parseEnid(ctx context.Context) error {
    parts := strings.Split(eNid, "=")
    eNid = parts[len(parts)-1]
    return nil
}

func loginTasks(username *string, password *string) chromedp.Tasks {
    return chromedp.Tasks{
        chromedp.Navigate("https://pgs-baas.bocajuniors.com.ar/baas/login.jsp?login_by=email"),
        chromedp.SetValue("input#email", *username),
        chromedp.SetValue("input#password", *password),
        chromedp.Click("span#btnEntrar button"),
        chromedp.WaitVisible("tbody div.site_box"),
        chromedp.Click("tbody div.site_box"),
        chromedp.WaitVisible("a[href=\"comprar.php\"]"),
    }
}

func gotoComprar() chromedp.Tasks {
    return chromedp.Tasks{
        chromedp.Navigate("https://soysocio.bocajuniors.com.ar/comprar.php"),
        chromedp.WaitVisible("div.contenidos div.columna3"),
        chromedp.AttributeValue("div.contenidos div.columna3 a", "href", &eNid, nil),
        chromedp.ActionFunc(parseEnid),
        chromedp.Click("div.contenidos div.columna3 a"),
        chromedp.WaitVisible("a#btnPlatea"),
        chromedp.Click("a#btnPlatea"),
        chromedp.WaitVisible("svg#statio"),
    }
}

func checkSeats(ctx context.Context) error {
    var seats []*cdp.Node
    var buttons []*cdp.Node
    for _, section := range sections {
        if !contains(ids, section.AttributeValue("id")) { continue }
        esNid := section.AttributeValue("data-nid")
        fmt.Printf("\nChecking section %v\n", section.AttributeValue("id"))
        chromedp.Run(ctx,
            chromedp.Navigate(fmt.Sprintf("https://soysocio.bocajuniors.com.ar/comprar_plano_asiento.php?eNid=%s&esNid=%s", eNid, esNid)),
            chromedp.WaitVisible("table.secmap"),
            chromedp.Nodes("table.secmap td.d", &seats, chromedp.AtLeast(0)),
        )
        fmt.Printf("\nChecking seats length\n")
        if len(seats) == 0 { chromedp.NavigateBack().Do(ctx); continue }
        chromedp.Run(ctx,
            chromedp.Click("table.secmap td.d"),
            chromedp.WaitVisible("span#ubicacionLugar"),
            chromedp.Nodes("a#btnReservar", &buttons, chromedp.AtLeast(0)),
        )
        fmt.Printf("\nChecking reserve button\n")
        if len(buttons) == 0 || buttons[0].AttributeValue("style") == "display: none;" { chromedp.NavigateBack().Do(ctx); continue }
        chromedp.Run(ctx,
            chromedp.ActionFunc(playSound),
            chromedp.Click("a#btnReservar"),
            chromedp.WaitVisible("svg#statio"),
        )
        break // TODO: lo hace con la primer seccion nomas
    }
    return nil
}

func checkSections() chromedp.Tasks {
    return chromedp.Tasks{
        chromedp.Reload(),
        chromedp.WaitVisible("svg#statio"),
        chromedp.Nodes("svg#statio g.enabled", &sections, chromedp.AtLeast(0)),
        chromedp.ActionFunc(printSections),
        chromedp.ActionFunc(checkSeats),
        chromedp.ActionFunc(func(ctx context.Context) error { return chromedp.Run(ctx, checkSections()) }),
    }
}

func main() {
    username, password := parseArgs()

    cancel := initSound()
    defer cancel()

    dir, err := os.MkdirTemp("", "chromedp-example")
    if err != nil { log.Fatal(err) }
    defer os.RemoveAll(dir)

    opts := append(chromedp.DefaultExecAllocatorOptions[:],
        chromedp.Flag("headless", false),
        chromedp.DisableGPU,
        chromedp.UserDataDir(dir),
    )

    allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
    defer cancel() // TODO: se puede sobreescribir cancel????

    taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
    defer cancel()

    err = chromedp.Run(taskCtx, loginTasks(username, password), gotoComprar(), checkSections())

    if err != nil { log.Fatal(err) }
}
