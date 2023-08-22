package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
	_ "embed"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
)

var primera      []string = []string{ "LID", "LIC", "LII", "LPD", "LPC", "LPI", "LD", "LV" }
var segunda      []string = []string{ "SDI", "SDD", "SBI", "SBC", "SBD", "SAI", "SAC", "SAD", "SCI", "SCD", "MI", "MC", "MD", "SMV" }
var tercera      []string = []string{ "I", "J", "F", "G", "H", "K" }
var torres       []string = []string{ "TS1", "TS2", "TS3", "TS4", "TN1", "TN2", "TN3", "TN4" }
var palcos       []string = []string{ "PLCVN", "PLCVS", "PLCPREF", "PPS1", "PPS2", "PPS3", "PPN1", "PPN2", "PPN3" }
var preferencial []string = []string{ "PRS1", "PRS2", "PRS3", "PRN1", "PRN2", "PRN3" }

var ids []string
var eNid string
var sections []*cdp.Node = []*cdp.Node{}
var previousSections []*cdp.Node = []*cdp.Node{}
var sameSections int = 1
var printed bool = false

//go:embed twitter.mp3
var audioBytes []byte
var streamer beep.StreamSeekCloser
var format beep.Format

func initSound() func() {
    var err error
    reader := io.NopCloser(bytes.NewReader(audioBytes))
    streamer, format, err = mp3.Decode(reader)
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
    if !printed {
        printed = true
    } else if !same {
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

func parseSections(selected string) (result []string) {
    if strings.Contains(selected, "1") { result = append(result, primera...) }
    if strings.Contains(selected, "2") { result = append(result, segunda...) }
    if strings.Contains(selected, "3") { result = append(result, tercera...) }
    if strings.Contains(selected, "t") { result = append(result, torres...) }
    if strings.Contains(selected, "p") { result = append(result, palcos...) }
    if strings.Contains(selected, "P") { result = append(result, preferencial...) }
    return result
}

func parseArgs() (username *string, password *string, selected *string) {
    if username = getArg("-u"); username == nil { username = getArg("--username") }
    if username == nil { log.Fatal("Username missing") }
    if password = getArg("-p"); password == nil { password = getArg("--password") }
    if password == nil { log.Fatal("Password missing") }
    if selected = getArg("-s"); selected == nil { selected = getArg("--sections") }
    if selected == nil { log.Fatal("Selected sections missing") }
    return username, password, selected
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
    var seats, buttons []*cdp.Node
    for _, section := range sections {
        if !contains(ids, section.AttributeValue("id")) { continue }
        esNid := section.AttributeValue("data-nid")
        chromedp.Run(ctx,
            chromedp.Navigate(fmt.Sprintf("https://soysocio.bocajuniors.com.ar/comprar_plano_asiento.php?eNid=%s&esNid=%s", eNid, esNid)),
            chromedp.WaitVisible("table.secmap"),
            chromedp.Nodes("table.secmap td.d", &seats, chromedp.AtLeast(0)),
        )
        if len(seats) == 0 { chromedp.NavigateBack().Do(ctx); continue }
        chromedp.Run(ctx,
            chromedp.Click("table.secmap td.d"),
            chromedp.WaitVisible("span#ubicacionLugar"),
            chromedp.Nodes("a#btnReservar", &buttons, chromedp.AtLeast(0)),
        )
        if len(buttons) == 0 || buttons[0].AttributeValue("style") == "display: none;" { chromedp.NavigateBack().Do(ctx); continue }
        chromedp.Run(ctx,
            chromedp.ActionFunc(playSound),
            chromedp.EvaluateAsDevTools("$('a#btnReservar').click()", nil),
            chromedp.WaitVisible("svg#statio"),
        )
        break
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
    username, password, selected := parseArgs()

    ids = parseSections(*selected)

    cancel := initSound()
    defer cancel()

    opts := append(
        chromedp.DefaultExecAllocatorOptions[:],
        chromedp.Flag("headless", false),
        chromedp.DisableGPU,
    )

    allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
    defer cancel() // TODO: se puede sobreescribir cancel????

    taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
    defer cancel()

    err := chromedp.Run(taskCtx, loginTasks(username, password))
    if err != nil { log.Fatal(err) }

    for {
        err = chromedp.Run(taskCtx, gotoComprar(), checkSections())
        // fmt.Printf("\nerror: %v", err.Error())
    }
}
