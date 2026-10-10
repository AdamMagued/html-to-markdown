package htmltomarkdown_test

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"testing"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"golang.org/x/net/html"
)

func ExampleConvertString() {
	input := `<strong>Bold Text</strong>`

	markdown, err := htmltomarkdown.ConvertString(input)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(markdown)
	// Output: **Bold Text**
}

func ExampleWithDomain() {
	input := `<img src="/assets/image.png" />`

	markdown, err := htmltomarkdown.ConvertString(
		input,
		// Provide a different domain for every convert call:
		converter.WithDomain("https://example.com"),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(markdown)
	// Output: ![](https://example.com/assets/image.png)
}

func ExampleConvertNode() {
	input := `<strong>Bold Text</strong>`

	doc, err := html.Parse(strings.NewReader(input))
	if err != nil {
		log.Fatal(err)
	}

	markdown, err := htmltomarkdown.ConvertNode(doc)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(markdown))
	// Output: **Bold Text**
}

func TestConvertString_WindowsCarriageReturn(t *testing.T) {
	testCases := []struct {
		desc string

		input    string
		expected string
	}{
		{
			desc: "just newlines",

			input:    "\r\n\r\n\r\n\r\n",
			expected: "",
		},
		{
			desc: "inside strong",

			input:    "<strong>Bold\r\n\r\n\r\n\r\nText</strong>",
			expected: "**Bold Text**",
		},
		{
			desc: "inside paragraph",

			input:    "<p>Some\r\n\r\n\r\n\r\nText</p>",
			expected: "Some Text",
		},
		{
			desc: "inside list",

			input:    "<ul><li>Some\r\n\r\n\r\n\r\nText</li></ul>",
			expected: "- Some Text",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			output, err := htmltomarkdown.ConvertString(tC.input)
			if err != nil {
				t.Fatal(err)
			}
			if output != tC.expected {
				t.Errorf("expected %q but got %q", tC.expected, output)
			}
		})
	}
}

func TestConvertString_LiteralBellCharacter(t *testing.T) {
	testCases := []struct {
		desc     string
		input    string
		expected string
	}{
		{
			desc:     "standalone bell character entity",
			input:    "<p>&#7;</p>",
			expected: "\ufffd",
		},
		{
			desc:     "standalone bell literal byte",
			input:    "<p>\a</p>",
			expected: "\ufffd",
		},
		{
			desc:     "bell at end of text",
			input:    "<p>a&#7;</p>",
			expected: "a\ufffd",
		},
		{
			desc:     "bell at end of text literal byte",
			input:    "<p>a\a</p>",
			expected: "a\ufffd",
		},
		{
			desc:     "bell at start of text",
			input:    "<p>&#7;a</p>",
			expected: "\ufffda",
		},
		{
			desc:     "bell adjacent to strong emphasis",
			input:    "<p>&#7;<strong>bold</strong></p>",
			expected: "\ufffd**bold**",
		},
		{
			desc:     "bell adjacent to strong emphasis literal byte",
			input:    "<p>\a<strong>bold</strong></p>",
			expected: "\ufffd**bold**",
		},
		{
			desc:     "bell adjacent to italic emphasis",
			input:    "<p>&#7;<em>italic</em></p>",
			expected: "\ufffd*italic*",
		},
		{
			desc:     "bell adjacent to italic emphasis literal byte",
			input:    "<p>\a<em>italic</em></p>",
			expected: "\ufffd*italic*",
		},
		{
			desc:     "bell inside strong element",
			input:    "<p><strong>&#7;bold</strong></p>",
			expected: "**\ufffdbold**",
		},
		{
			desc:     "bell inside italic element",
			input:    "<p><em>&#7;italic</em></p>",
			expected: "*\ufffditalic*",
		},
		{
			desc:     "bell adjacent to link",
			input:    `<p>&#7;<a href="https://example.com">link</a></p>`,
			expected: "\ufffd[link](https://example.com)",
		},
		{
			desc:     "bell inside inline code",
			input:    "<p><code>&#7;</code></p>",
			expected: "`\ufffd`",
		},
		{
			desc:     "multiple bell characters",
			input:    "<p>&#7;&#7;&#7;</p>",
			expected: "\ufffd\ufffd\ufffd",
		},
		{
			desc:     "multiple bell characters literal bytes",
			input:    "<p>\a\a\a</p>",
			expected: "\ufffd\ufffd\ufffd",
		},
		{
			desc:     "bell adjacent to heading",
			input:    "<h1>&#7;Heading</h1>",
			expected: "# \ufffdHeading",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			output, err := htmltomarkdown.ConvertString(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			if output != tc.expected {
				t.Errorf("expected %q but got %q", tc.expected, output)
			}
		})
	}
}

func TestDataRaceDetector(t *testing.T) {
	conv := converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(),
		),
	)

	input := `<i>italic text</i>`

	var wg sync.WaitGroup

	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func() {
			conv.Register.EscapedChar('~')
			conv.Register.UnEscaper(
				func(chars []byte, index int) int { return -1 },
				converter.PriorityStandard,
			)
			conv.Register.PreRenderer(
				func(ctx converter.Context, doc *html.Node) {},
				converter.PriorityStandard,
			)
			conv.Register.TextTransformer(
				func(ctx converter.Context, content string) string { return content },
				converter.PriorityStandard,
			)
			conv.Register.Renderer(
				func(ctx converter.Context, w converter.Writer, n *html.Node) converter.RenderStatus {
					return converter.RenderTryNext
				},
				converter.PriorityStandard,
			)
			conv.Register.PostRenderer(
				func(ctx converter.Context, content []byte) []byte {
					return content
				},
				converter.PriorityStandard,
			)

			conv.Register.TagType("script", converter.TagTypeBlock, converter.PriorityStandard)

			output, err := conv.ConvertString(input, converter.WithDomain("example.com"))
			if err != nil {
				t.Error(err)
			}
			_ = output

			output2, err := conv.ConvertString(input)
			if err != nil {
				t.Error(err)
			}
			_ = output2

			wg.Done()
		}()
	}
	wg.Wait()
}
