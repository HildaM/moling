// Copyright 2025 CFC4N <cfc4n.cs@gmail.com>. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Repository: https://github.com/gojue/moling

// Package services provides a set of services for the MoLing application.
package browser

import (
	"fmt"
	"os"
	"path/filepath"
)

const BrowserPromptDefault = `
You are an AI-powered browser automation assistant capable of performing a wide range of web interactions and debugging tasks. Your capabilities include:

1. **Navigation**: Navigate to any specified URL to load web pages.

2. **Screenshot Capture**: Take full-page screenshots or capture specific elements using CSS selectors, with customizable dimensions (default: 1700x1100 pixels).

3. **Element Interaction**:
   - Click on elements identified by CSS selectors
   - Hover over specified elements
   - Fill input fields with provided values
   - Select options in dropdown menus

4. **JavaScript Execution**:
   - Run arbitrary JavaScript code in the browser context
   - Evaluate scripts and return results

5. **Debugging Tools**:
   - Enable/disable JavaScript debugging mode
   - Set breakpoints at specific script locations (URL + line number + optional column/condition)
   - Remove existing breakpoints by ID
   - Pause and resume script execution
   - Retrieve current call stack when paused

For all actions requiring element selection, you must use precise CSS selectors. When capturing screenshots, you can specify either the entire page or target specific elements. For debugging operations, you can precisely control execution flow and inspect runtime behavior.

Please provide clear instructions including:
- The specific action you want performed
- Required parameters (URLs, selectors, values, etc.)
- Any optional parameters (dimensions, conditions, etc.)
- Expected outcomes where relevant

You should confirm actions before execution when dealing with sensitive operations or destructive commands. Report back with clear status updates, success/failure indicators, and any relevant output or captured data.
`

type BrowserConfig struct {
	PromptFile           string `json:"prompt_file"` // PromptFile is the prompt file for the browser.
	prompt               string
	Headless             bool   `json:"headless"`
	Timeout              int    `json:"timeout"`
	Proxy                string `json:"proxy"`
	UserAgent            string `json:"user_agent"`
	DefaultLanguage      string `json:"default_language"`
	URLTimeout           int    `json:"url_timeout"`            // URLTimeout is the timeout for loading a URL. time.Second
	SelectorQueryTimeout int    `json:"selector_query_timeout"` // SelectorQueryTimeout is the timeout for CSS selector queries. time.Second
	DataPath             string `json:"data_path"`              // DataPath is the path to the data directory.
	BrowserDataPath      string `json:"browser_data_path"`      // BrowserDataPath is the path to the browser data directory.
}

func (cfg *BrowserConfig) Check() error {
	cfg.prompt = BrowserPromptDefault
	if cfg.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than 0")
	}
	if cfg.URLTimeout <= 0 {
		return fmt.Errorf("URL timeout must be greater than 0")
	}
	if cfg.SelectorQueryTimeout <= 0 {
		return fmt.Errorf("selector Query timeout must be greater than 0")
	}
	if cfg.PromptFile != "" {
		read, err := os.ReadFile(cfg.PromptFile)
		if err != nil {
			return fmt.Errorf("failed to read prompt file:%s, error: %v", cfg.PromptFile, err)
		}
		cfg.prompt = string(read)
	}
	return nil
}

// NewBrowserConfig creates a new BrowserConfig with default values.
// TODO 待配置化
func NewBrowserConfig() *BrowserConfig {
	return &BrowserConfig{
		Headless:             false,
		Timeout:              30,
		URLTimeout:           10,
		SelectorQueryTimeout: 20,
		UserAgent:            "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36",
		DefaultLanguage:      "en-US",
		DataPath:             filepath.Join(os.TempDir(), ".moling", "data"),
	}
}
