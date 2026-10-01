package main

import (
	"context"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bendahl/uinput"
)

var keyCodeMap = map[string]int{
	"esc":              uinput.KeyEsc,
	"1":                uinput.Key1,
	"2":                uinput.Key2,
	"3":                uinput.Key3,
	"4":                uinput.Key4,
	"5":                uinput.Key5,
	"6":                uinput.Key6,
	"7":                uinput.Key7,
	"8":                uinput.Key8,
	"9":                uinput.Key9,
	"0":                uinput.Key0,
	"minus":            uinput.KeyMinus,
	"equal":            uinput.KeyEqual,
	"backspace":        uinput.KeyBackspace,
	"tab":              uinput.KeyTab,
	"q":                uinput.KeyQ,
	"w":                uinput.KeyW,
	"e":                uinput.KeyE,
	"r":                uinput.KeyR,
	"t":                uinput.KeyT,
	"y":                uinput.KeyY,
	"u":                uinput.KeyU,
	"i":                uinput.KeyI,
	"o":                uinput.KeyO,
	"p":                uinput.KeyP,
	"leftbrace":        uinput.KeyLeftbrace,
	"rightbrace":       uinput.KeyRightbrace,
	"enter":            uinput.KeyEnter,
	"leftctrl":         uinput.KeyLeftctrl,
	"a":                uinput.KeyA,
	"s":                uinput.KeyS,
	"d":                uinput.KeyD,
	"f":                uinput.KeyF,
	"g":                uinput.KeyG,
	"h":                uinput.KeyH,
	"j":                uinput.KeyJ,
	"k":                uinput.KeyK,
	"l":                uinput.KeyL,
	"semicolon":        uinput.KeySemicolon,
	"apostrophe":       uinput.KeyApostrophe,
	"grave":            uinput.KeyGrave,
	"leftshift":        uinput.KeyLeftshift,
	"backslash":        uinput.KeyBackslash,
	"z":                uinput.KeyZ,
	"x":                uinput.KeyX,
	"c":                uinput.KeyC,
	"v":                uinput.KeyV,
	"b":                uinput.KeyB,
	"n":                uinput.KeyN,
	"m":                uinput.KeyM,
	"comma":            uinput.KeyComma,
	"dot":              uinput.KeyDot,
	"slash":            uinput.KeySlash,
	"rightshift":       uinput.KeyRightshift,
	"kpasterisk":       uinput.KeyKpasterisk,
	"leftalt":          uinput.KeyLeftalt,
	"space":            uinput.KeySpace,
	"capslock":         uinput.KeyCapslock,
	"f1":               uinput.KeyF1,
	"f2":               uinput.KeyF2,
	"f3":               uinput.KeyF3,
	"f4":               uinput.KeyF4,
	"f5":               uinput.KeyF5,
	"f6":               uinput.KeyF6,
	"f7":               uinput.KeyF7,
	"f8":               uinput.KeyF8,
	"f9":               uinput.KeyF9,
	"f10":              uinput.KeyF10,
	"numlock":          uinput.KeyNumlock,
	"scrolllock":       uinput.KeyScrolllock,
	"kp7":              uinput.KeyKp7,
	"kp8":              uinput.KeyKp8,
	"kp9":              uinput.KeyKp9,
	"kpminus":          uinput.KeyKpminus,
	"kp4":              uinput.KeyKp4,
	"kp5":              uinput.KeyKp5,
	"kp6":              uinput.KeyKp6,
	"kpplus":           uinput.KeyKpplus,
	"kp1":              uinput.KeyKp1,
	"kp2":              uinput.KeyKp2,
	"kp3":              uinput.KeyKp3,
	"kp0":              uinput.KeyKp0,
	"kpdot":            uinput.KeyKpdot,
	"zenkakuhankaku":   uinput.KeyZenkakuhankaku,
	"102nd":            uinput.Key102Nd,
	"f11":              uinput.KeyF11,
	"f12":              uinput.KeyF12,
	"ro":               uinput.KeyRo,
	"katakana":         uinput.KeyKatakana,
	"hiragana":         uinput.KeyHiragana,
	"henkan":           uinput.KeyHenkan,
	"katakanahiragana": uinput.KeyKatakanahiragana,
	"muhenkan":         uinput.KeyMuhenkan,
	"kpjpcomma":        uinput.KeyKpjpcomma,
	"kpenter":          uinput.KeyKpenter,
	"rightctrl":        uinput.KeyRightctrl,
	"kpslash":          uinput.KeyKpslash,
	"sysrq":            uinput.KeySysrq,
	"rightalt":         uinput.KeyRightalt,
	"linefeed":         uinput.KeyLinefeed,
	"home":             uinput.KeyHome,
	"up":               uinput.KeyUp,
	"pageup":           uinput.KeyPageup,
	"left":             uinput.KeyLeft,
	"right":            uinput.KeyRight,
	"end":              uinput.KeyEnd,
	"down":             uinput.KeyDown,
	"pagedown":         uinput.KeyPagedown,
	"insert":           uinput.KeyInsert,
	"delete":           uinput.KeyDelete,
	"macro":            uinput.KeyMacro,
	"mute":             uinput.KeyMute,
	"volumedown":       uinput.KeyVolumedown,
	"volumeup":         uinput.KeyVolumeup,
	"power":            uinput.KeyPower,
	"kpequal":          uinput.KeyKpequal,
	"kpplusminus":      uinput.KeyKpplusminus,
	"pause":            uinput.KeyPause,
	"scale":            uinput.KeyScale,
	"kpcomma":          uinput.KeyKpcomma,
	"hangeul":          uinput.KeyHangeul,
	"hanja":            uinput.KeyHanja,
	"yen":              uinput.KeyYen,
	"leftmeta":         uinput.KeyLeftmeta,
	"rightmeta":        uinput.KeyRightmeta,
	"compose":          uinput.KeyCompose,
	"stop":             uinput.KeyStop,
	"again":            uinput.KeyAgain,
	"props":            uinput.KeyProps,
	"undo":             uinput.KeyUndo,
	"front":            uinput.KeyFront,
	"copy":             uinput.KeyCopy,
	"open":             uinput.KeyOpen,
	"paste":            uinput.KeyPaste,
	"find":             uinput.KeyFind,
	"cut":              uinput.KeyCut,
	"help":             uinput.KeyHelp,
	"menu":             uinput.KeyMenu,
	"calc":             uinput.KeyCalc,
	"setup":            uinput.KeySetup,
	"sleep":            uinput.KeySleep,
	"wakeup":           uinput.KeyWakeup,
	"file":             uinput.KeyFile,
	"sendfile":         uinput.KeySendfile,
	"deletefile":       uinput.KeyDeletefile,
	"xfer":             uinput.KeyXfer,
	"prog1":            uinput.KeyProg1,
	"prog2":            uinput.KeyProg2,
	"www":              uinput.KeyWww,
	"msdos":            uinput.KeyMsdos,
	"coffee":           uinput.KeyCoffee,
	"direction":        uinput.KeyDirection,
	"cyclewindows":     uinput.KeyCyclewindows,
	"mail":             uinput.KeyMail,
	"bookmarks":        uinput.KeyBookmarks,
	"computer":         uinput.KeyComputer,
	"back":             uinput.KeyBack,
	"forward":          uinput.KeyForward,
	"closecd":          uinput.KeyClosecd,
	"ejectcd":          uinput.KeyEjectcd,
	"ejectclosecd":     uinput.KeyEjectclosecd,
	"nextsong":         uinput.KeyNextsong,
	"playpause":        uinput.KeyPlaypause,
	"previoussong":     uinput.KeyPrevioussong,
	"stopcd":           uinput.KeyStopcd,
	"record":           uinput.KeyRecord,
	"rewind":           uinput.KeyRewind,
	"phone":            uinput.KeyPhone,
	"iso":              uinput.KeyIso,
	"config":           uinput.KeyConfig,
	"homepage":         uinput.KeyHomepage,
	"refresh":          uinput.KeyRefresh,
	"exit":             uinput.KeyExit,
	"move":             uinput.KeyMove,
	"edit":             uinput.KeyEdit,
	"scrollup":         uinput.KeyScrollup,
	"scrolldown":       uinput.KeyScrolldown,
	"kpleftparen":      uinput.KeyKpleftparen,
	"kprightparen":     uinput.KeyKprightparen,
	"new":              uinput.KeyNew,
	"redo":             uinput.KeyRedo,
	"f13":              uinput.KeyF13,
	"f14":              uinput.KeyF14,
	"f15":              uinput.KeyF15,
	"f16":              uinput.KeyF16,
	"f17":              uinput.KeyF17,
	"f18":              uinput.KeyF18,
	"f19":              uinput.KeyF19,
	"f20":              uinput.KeyF20,
	"f21":              uinput.KeyF21,
	"f22":              uinput.KeyF22,
	"f23":              uinput.KeyF23,
	"f24":              uinput.KeyF24,
	"playcd":           uinput.KeyPlaycd,
	"pausecd":          uinput.KeyPausecd,
	"prog3":            uinput.KeyProg3,
	"prog4":            uinput.KeyProg4,
	"dashboard":        uinput.KeyDashboard,
	"suspend":          uinput.KeySuspend,
	"close":            uinput.KeyClose,
	"play":             uinput.KeyPlay,
	"fastforward":      uinput.KeyFastforward,
	"bassboost":        uinput.KeyBassboost,
	"print":            uinput.KeyPrint,
	"hp":               uinput.KeyHp,
	"camera":           uinput.KeyCamera,
	"sound":            uinput.KeySound,
	"question":         uinput.KeyQuestion,
	"email":            uinput.KeyEmail,
	"chat":             uinput.KeyChat,
	"search":           uinput.KeySearch,
	"connect":          uinput.KeyConnect,
	"finance":          uinput.KeyFinance,
	"sport":            uinput.KeySport,
	"shop":             uinput.KeyShop,
	"alterase":         uinput.KeyAlterase,
	"cancel":           uinput.KeyCancel,
	"brightnessdown":   uinput.KeyBrightnessdown,
	"brightnessup":     uinput.KeyBrightnessup,
	"media":            uinput.KeyMedia,
	"switchvideomode":  uinput.KeySwitchvideomode,
	"kbdillumtoggle":   uinput.KeyKbdillumtoggle,
	"kbdillumdown":     uinput.KeyKbdillumdown,
	"kbdillumup":       uinput.KeyKbdillumup,
	"send":             uinput.KeySend,
	"reply":            uinput.KeyReply,
	"forwardmail":      uinput.KeyForwardmail,
	"save":             uinput.KeySave,
	"documents":        uinput.KeyDocuments,
	"battery":          uinput.KeyBattery,
	"bluetooth":        uinput.KeyBluetooth,
	"wlan":             uinput.KeyWlan,
	"uwb":              uinput.KeyUwb,
	"unknown":          uinput.KeyUnknown,
	"videonext":        uinput.KeyVideoNext,
	"videoprev":        uinput.KeyVideoPrev,
	"brightnesscycle":  uinput.KeyBrightnessCycle,
	"brightnesszero":   uinput.KeyBrightnessZero,
	"displayoff":       uinput.KeyDisplayOff,
	"wimax":            uinput.KeyWimax,
	"rfkill":           uinput.KeyRfkill,
	"micmute":          uinput.KeyMicmute,
}

var virtualKeyboard *uinputKeyboard

var keyCodeNames = buildKeyCodeNames()

func buildKeyCodeNames() map[int]string {
	names := make(map[int]string, len(keyCodeMap))
	for name, code := range keyCodeMap {
		names[code] = name
	}
	return names
}

func keyName(code int) string {
	if name, ok := keyCodeNames[code]; ok {
		return name
	}
	return strconv.Itoa(code)
}

func keyNames(codes []int) string {
	parts := make([]string, 0, len(codes))
	for _, code := range codes {
		parts = append(parts, keyName(code))
	}
	return strings.Join(parts, " ")
}

func resolveKeyCode(name string) int {
	if name == "" {
		return 0
	}
	code, ok := keyCodeMap[strings.ToLower(name)]
	if !ok {
		log.Printf("unknown key %q", name)
		return 0
	}
	return code
}

func pressKeyOnce(code int) {
	if virtualKeyboard == nil {
		return
	}
	remapState.markInjected(code)
	if err := virtualKeyboard.TapKey(code); err != nil {
		log.Printf("uinput key tap failed: %v", err)
	}
}

func pressKeyPeriodically(ctx context.Context, wg *sync.WaitGroup, action KeyAction) {
	defer wg.Done()
	code, ok := keyCodeMap[strings.ToLower(action.Key)]
	if !ok {
		return
	}

	ticker := time.NewTicker(time.Duration(action.Interval * float64(time.Second)))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if pauseState.paused() {
				continue
			}
			pressKeyOnce(code)
		}
	}
}
