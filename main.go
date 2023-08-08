package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
)

var emptySections int

var ids []string = []string{ "I", "J", "F", "G", "H", "K" }

var soundFile *os.File
var streamer beep.StreamSeekCloser
var format beep.Format

func initSound() {
    var err error
    soundFile, err = os.Open("./twitter.mp3")
    if err != nil { log.Fatal(err) }
    streamer, format, err = mp3.Decode(soundFile)
    if err != nil { log.Fatal(err) }
    // defer streamer.Close()
    speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
}

func playSound() func(context.Context) error {
    return func(ctx context.Context) error {
        speaker.Play(streamer)
        return nil
    }
}

func printSections(sections []map[string]string) {
    if len(sections) > 0 {
        fmt.Printf("\nsecciones: %v", sections)
        emptySections = 0
    } else if emptySections == 0 {
        fmt.Printf("\nsecciones: %v x %v", sections, emptySections)
        emptySections++
    } else {
        fmt.Printf("\rsecciones: %v x %v", sections, emptySections)
        emptySections++
    }
}

func logg(text string) func(context.Context) error {
    return func(ctx context.Context) error {
        fmt.Println("[LOG]", text)
        return nil
    }
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

func parseEnid(eNid *string) func(context.Context) error {
    return func(ctx context.Context) error {
        parts := strings.Split(*eNid, "=")
        *eNid = parts[len(parts)-1]
        return nil
    }
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

func gotoComprar(eNid *string) chromedp.Tasks {
    return chromedp.Tasks{
        chromedp.Navigate("https://soysocio.bocajuniors.com.ar/comprar.php"),
        chromedp.WaitVisible("div.contenidos div.columna3"),
        chromedp.AttributeValue("div.contenidos div.columna3 a", "href", eNid, nil),
        chromedp.ActionFunc(parseEnid(eNid)),
        chromedp.Click("div.contenidos div.columna3 a"),
        chromedp.WaitVisible("a#btnPlatea"),
        chromedp.Click("a#btnPlatea"),
        chromedp.WaitVisible("svg#statio"),
    }
}

func checkSeats(eNid string, sections []map[string]string) func(context.Context) error {
    return func (ctx context.Context) error {
        for _, section := range sections {
            if !contains(ids, section["id"]) { continue }
            esNid := section["data-nid"] // "68020"
            chromedp.Run(ctx,
                chromedp.Navigate(fmt.Sprintf("https://soysocio.bocajuniors.com.ar/comprar_plano_asiento.php?eNid=%s&esNid=%s", eNid, esNid)),
                chromedp.WaitVisible("table.secmap"),
                chromedp.Click("table.secmap td.d"),
                chromedp.WaitVisible("span#ubicacionLugar"),
                chromedp.Click("a#btnReservar"),
                chromedp.WaitVisible("svg#statio"),
            )
            break // TODO: lo hace con la primer seccion nomas
        }
        return nil
    }
}

// func checkSections(eNid string) chromedp.Tasks {
//     sections := []map[string]string{}
//     return chromedp.Tasks{
//         chromedp.WaitVisible("svg#statio"),
//         chromedp.Sleep(200 * time.Millisecond),
//         chromedp.Reload(),
//         chromedp.WaitVisible("svg#statio"),
//         chromedp.AttributesAll("div#divMap switch g.enabled", &sections, chromedp.AtLeast(0)),
//         chromedp.ActionFunc(logg(fmt.Sprintf("secciones: %v", sections))),
//         chromedp.ActionFunc(checkSeats(eNid, sections)), chromedp.ActionFunc(func(ctx context.Context) error { return chromedp.Run(ctx, checkSections(eNid)) }),
//     }
// }

func main() {
    username, password := parseArgs()

    initSound()
    defer streamer.Close()

    dir, err := os.MkdirTemp("", "chromedp-example")
    if err != nil { log.Fatal(err) }
    defer os.RemoveAll(dir)

    opts := append(chromedp.DefaultExecAllocatorOptions[:],
        chromedp.Flag("headless", false),
        chromedp.DisableGPU,
        chromedp.UserDataDir(dir),
    )

    allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
    defer cancel()

    taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
    defer cancel()

    eNid := ""

    // if err = chromedp.Run(taskCtx, loginTasks(username, password), gotoComprar(&eNid), checkSections(eNid)); err != nil { log.Fatal(err) }

    if err = chromedp.Run(taskCtx, loginTasks(username, password), gotoComprar(&eNid)); err != nil { log.Fatal(err) }

    sections := []map[string]string{}
    for {
        chromedp.Run(taskCtx,
            chromedp.WaitVisible("svg#statio"),
            chromedp.AttributesAll("div#divMap switch g.enabled", &sections, chromedp.AtLeast(0)),
        )

        printSections(sections)

        if len(sections) > 0 {
            for _, section := range sections {
                if !contains(ids, section["id"]) { continue }
                esNid := section["data-nid"] // "68020"
                chromedp.Run(taskCtx,
                    chromedp.Navigate(fmt.Sprintf("https://soysocio.bocajuniors.com.ar/comprar_plano_asiento.php?eNid=%s&esNid=%s", eNid, esNid)),
                    chromedp.WaitVisible("table.secmap"),
                    chromedp.Click("table.secmap td.d"),
                    chromedp.WaitVisible("span#ubicacionLugar"),
                    chromedp.Click("a#btnReservar"),
                    chromedp.ActionFunc(playSound()),
                    chromedp.WaitVisible("svg#statio"),
                )
                break // TODO lo hace con el primero
            }
        }

        sections = []map[string]string{}
        time.Sleep(200 * time.Millisecond)
        chromedp.Reload().Do(taskCtx)
    }
}
