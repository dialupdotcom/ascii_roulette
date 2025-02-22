package ui

import (
	"image"
	"regexp"
	"unicode"
	"unicode/utf8"

	"github.com/dialup-inc/ascii/term"
)

func StateReducer(s State, event Event) State {
	s.Image = imageReducer(s.Image, event)
	s.ChatActive = chatActiveReducer(s.ChatActive, event)
	s.Input = inputReducer(s.Input, s.ChatActive, event)
	s.Messages = messagesReducer(s.Messages, event)
	s.Page = pageReducer(s.Page, event)
	s.WinSize = winSizeReducer(s.WinSize, event)
	s.HelpOn = helpOnReducer(s.HelpOn, event)

	return s
}

func chatActiveReducer(s bool, event Event) bool {
	switch event.(type) {
	case DataOpenedEvent:
		return true
	case ConnEndedEvent:
		return false
	default:
		return s
	}
}

func helpOnReducer(s bool, event Event) bool {
	switch event.(type) {
	case ToggleHelpEvent:
		return !s
	case SkipEvent:
		return false
	case SentMessageEvent:
		return false
	default:
		return s
	}
}

func pageReducer(s Page, event Event) Page {
	switch e := event.(type) {
	case SetPageEvent:
		return Page(e)
	default:
		return s
	}
}

func winSizeReducer(s term.WinSize, event Event) term.WinSize {
	switch e := event.(type) {
	case ResizeEvent:
		return term.WinSize(e)
	default:
		return s
	}
}

var ansiRegex = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

func inputReducer(s string, chatActive bool, event Event) string {
	switch e := event.(type) {
	case ConnStartedEvent:
		return ""

	case KeypressEvent:
		if !chatActive {
			return s
		}

		s += string(e)

		// Strip ANSI escape codes
		s = ansiRegex.ReplaceAllString(s, "")

		// Strip unprintable characters
		var printable []rune
		for _, r := range s {
			if !unicode.IsPrint(r) {
				continue
			}
			printable = append(printable, r)
		}
		s = string(printable)

		return s

	case BackspaceEvent:
		if !chatActive {
			return s
		}

		if len(s) == 0 {
			return s
		}
		_, sz := utf8.DecodeLastRuneInString(s)
		return s[:len(s)-sz]

	case SentMessageEvent:
		return ""

	default:
		return s
	}
}

func imageReducer(s image.Image, event Event) image.Image {
	switch e := event.(type) {
	case FrameEvent:
		return image.Image(e)

	case SetPageEvent:
		return nil

	case SkipEvent:
		return nil

	default:
		return s
	}
}

func messagesReducer(s []Message, event Event) []Message {
	switch e := event.(type) {
	case SentMessageEvent:
		return append(s, Message{
			Type: MessageTypeOutgoing,
			User: "You",
			Text: string(e),
		})

	case ReceivedChatEvent:
		return append(s, Message{
			Type: MessageTypeIncoming,
			User: "Them",
			Text: string(e),
		})

	case ConnEndedEvent:
		var msg Message

		switch e.Reason {

		// Error handler will catch this
		case EndConnSetupError:
			return s

		// We ignore match errors
		case EndConnMatchError:
			return s

		case EndConnNormal:
			msg = Message{
				Type: MessageTypeInfo,
				Text: "Skipping...",
			}

		case EndConnTimedOut:
			msg = Message{
				Type: MessageTypeError,
				Text: "Connection timed out.",
			}

		case EndConnDisconnected:
			msg = Message{
				Type: MessageTypeError,
				Text: "Lost connection.",
			}

		case EndConnGone:
			msg = Message{
				Type: MessageTypeInfo,
				Text: "Your partner left the chat.",
			}
		}

		return append(s, msg)

	case ConnStartedEvent:
		return append(s, Message{
			Type: MessageTypeInfo,
			Text: "Connected",
		})

	case LogEvent:
		mtype := MessageTypeInfo
		if e.Level == LogLevelError {
			mtype = MessageTypeError
		}

		return append(s, Message{
			Type: mtype,
			Text: e.Text,
		})

	default:
		return s
	}
}
