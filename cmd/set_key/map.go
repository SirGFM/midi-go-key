package main

import (
	"github.com/micmonay/keybd_event"
)

// Maps each key name to its value.
var keyNameToInt = map[string]int{
	"UP":    keybd_event.VK_UP,
	"DOWN":  keybd_event.VK_DOWN,
	"LEFT":  keybd_event.VK_LEFT,
	"RIGHT": keybd_event.VK_RIGHT,

	"ESC": keybd_event.VK_ESC,
	"1":   keybd_event.VK_1,
	"2":   keybd_event.VK_2,
	"3":   keybd_event.VK_3,
	"4":   keybd_event.VK_4,
	"5":   keybd_event.VK_5,
	"6":   keybd_event.VK_6,
	"7":   keybd_event.VK_7,
	"8":   keybd_event.VK_8,
	"9":   keybd_event.VK_9,
	"0":   keybd_event.VK_0,
	"Q":   keybd_event.VK_Q,
	"W":   keybd_event.VK_W,
	"E":   keybd_event.VK_E,
	"R":   keybd_event.VK_R,
	"T":   keybd_event.VK_T,
	"Y":   keybd_event.VK_Y,
	"U":   keybd_event.VK_U,
	"I":   keybd_event.VK_I,
	"O":   keybd_event.VK_O,
	"P":   keybd_event.VK_P,
	"A":   keybd_event.VK_A,
	"S":   keybd_event.VK_S,
	"D":   keybd_event.VK_D,
	"F":   keybd_event.VK_F,
	"G":   keybd_event.VK_G,
	"H":   keybd_event.VK_H,
	"J":   keybd_event.VK_J,
	"K":   keybd_event.VK_K,
	"L":   keybd_event.VK_L,
	"Z":   keybd_event.VK_Z,
	"X":   keybd_event.VK_X,
	"C":   keybd_event.VK_C,
	"V":   keybd_event.VK_V,
	"B":   keybd_event.VK_B,
	"N":   keybd_event.VK_N,
	"M":   keybd_event.VK_M,
	"F1":  keybd_event.VK_F1,
	"F2":  keybd_event.VK_F2,
	"F3":  keybd_event.VK_F3,
	"F4":  keybd_event.VK_F4,
	"F5":  keybd_event.VK_F5,
	"F6":  keybd_event.VK_F6,
	"F7":  keybd_event.VK_F7,
	"F8":  keybd_event.VK_F8,
	"F9":  keybd_event.VK_F9,
	"F10": keybd_event.VK_F10,
	"F11": keybd_event.VK_F11,
	"F12": keybd_event.VK_F12,

	"NUMLOCK":    keybd_event.VK_NUMLOCK,
	"SCROLLLOCK": keybd_event.VK_SCROLLLOCK,
	"RESERVED":   keybd_event.VK_RESERVED,
	"MINUS":      keybd_event.VK_MINUS,
	"EQUAL":      keybd_event.VK_EQUAL,
	"BACKSPACE":  keybd_event.VK_BACKSPACE,
	"TAB":        keybd_event.VK_TAB,
	"LEFTBRACE":  keybd_event.VK_LEFTBRACE,
	"RIGHTBRACE": keybd_event.VK_RIGHTBRACE,
	"ENTER":      keybd_event.VK_ENTER,
	"SEMICOLON":  keybd_event.VK_SEMICOLON,
	"APOSTROPHE": keybd_event.VK_APOSTROPHE,
	"GRAVE":      keybd_event.VK_GRAVE,
	"BACKSLASH":  keybd_event.VK_BACKSLASH,
	"COMMA":      keybd_event.VK_COMMA,
	"DOT":        keybd_event.VK_DOT,
	"SLASH":      keybd_event.VK_SLASH,
	"SPACE":      keybd_event.VK_SPACE,
	"CAPSLOCK":   keybd_event.VK_CAPSLOCK,

	"KP0":        keybd_event.VK_KP0,
	"KP1":        keybd_event.VK_KP1,
	"KP2":        keybd_event.VK_KP2,
	"KP3":        keybd_event.VK_KP3,
	"KP4":        keybd_event.VK_KP4,
	"KP5":        keybd_event.VK_KP5,
	"KP6":        keybd_event.VK_KP6,
	"KP7":        keybd_event.VK_KP7,
	"KP8":        keybd_event.VK_KP8,
	"KP9":        keybd_event.VK_KP9,
	"KPMINUS":    keybd_event.VK_KPMINUS,
	"KPPLUS":     keybd_event.VK_KPPLUS,
	"KPDOT":      keybd_event.VK_KPDOT,
	"KPASTERISK": keybd_event.VK_KPASTERISK,

	"HOME":     keybd_event.VK_HOME,
	"PAGEUP":   keybd_event.VK_PAGEUP,
	"END":      keybd_event.VK_END,
	"PAGEDOWN": keybd_event.VK_PAGEDOWN,
	"INSERT":   keybd_event.VK_INSERT,
	"DELETE":   keybd_event.VK_DELETE,
	"PAUSE":    keybd_event.VK_PAUSE,

	"F13": keybd_event.VK_F13,
	"F14": keybd_event.VK_F14,
	"F15": keybd_event.VK_F15,
	"F16": keybd_event.VK_F16,
	"F17": keybd_event.VK_F17,
	"F18": keybd_event.VK_F18,
	"F19": keybd_event.VK_F19,
	"F20": keybd_event.VK_F20,
	"F21": keybd_event.VK_F21,
	"F22": keybd_event.VK_F22,
	"F23": keybd_event.VK_F23,
	"F24": keybd_event.VK_F24,
}
