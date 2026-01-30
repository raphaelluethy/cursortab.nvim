# cursortab.nvim

A Neovim plugin that provides AI-powered edit completions and cursor predictions
using the Sweep hosted API.

> [!WARNING]
>
> **This is an early-stage, beta project.** Expect bugs, incomplete features,
> and breaking changes.

<p align="center">
    <img src="assets/demo.gif" width="600">
</p>

<!-- mtoc-start -->

* [Requirements](#requirements)
* [Installation](#installation)
  * [Using lazy.nvim](#using-lazynvim)
  * [Using packer.nvim](#using-packernvim)
* [Configuration](#configuration)
* [Usage](#usage)
  * [Commands](#commands)
* [Development](#development)
  * [Build](#build)
  * [Test](#test)
* [FAQ](#faq)
* [Contributing](#contributing)
* [License](#license)

<!-- mtoc-end -->

## Requirements

- Go 1.24.2+ (for building the server component)
- Neovim 0.8+ (for the plugin)

## Installation

### Using [lazy.nvim](https://github.com/folke/lazy.nvim)

```lua
{
  "leonardcser/cursortab.nvim",
  -- version = "*",  -- Use latest tagged version for more stability
  build = "cd server && go build",
  config = function()
    require("cursortab").setup()
  end,
}
```

### Using [packer.nvim](https://github.com/wbthomason/packer.nvim)

```lua
use {
  "leonardcser/cursortab.nvim",
  -- tag = "*",  -- Use latest tagged version for more stability
  run = "cd server && go build",
  config = function()
    require("cursortab").setup()
  end
}
```

## Configuration

Before using the plugin, you need to set up authentication with the Sweep API:

```bash
export SWEEP_AI_TOKEN="your-api-key-here"
```

Then configure the plugin:

```lua
require("cursortab").setup({
  enabled = true,
  log_level = "info",  -- "trace", "debug", "info", "warn", "error"

  ui = {
    colors = {
      deletion = "#4f2f2f",      -- Background color for deletions
      addition = "#394f2f",      -- Background color for additions
      modification = "#282e38",  -- Background color for modifications
      completion = "#80899c",    -- Foreground color for completions
    },
    jump = {
      symbol = "",              -- Symbol shown for jump points
      text = " TAB ",            -- Text displayed after jump symbol
      show_distance = true,      -- Show line distance for off-screen jumps
      bg_color = "#373b45",      -- Jump text background color
      fg_color = "#bac1d1",      -- Jump text foreground color
    },
  },

  behavior = {
    idle_completion_delay = 50,  -- Delay in ms after idle to trigger completion (-1 to disable)
    text_change_debounce = 50,   -- Debounce in ms after text change to trigger completion
    cursor_prediction = {
      enabled = true,            -- Show jump indicators after completions
      auto_advance = true,       -- When no changes, show cursor jump to last line
      proximity_threshold = 2,   -- Min lines apart to show cursor jump (0 to disable)
    },
  },

  provider = {
    type = "sweep",
    url = "https://autocomplete.sweep.dev",
    temperature = 0.0,
    max_tokens = 512,
    top_k = 50,
    completion_timeout = 5000,
    max_diff_history_tokens = 512,
    api_key = nil,                -- API key (nil to use env var)
    api_key_env = "SWEEP_AI_TOKEN",
  },

  debug = {
    immediate_shutdown = false,  -- Shutdown daemon immediately when no clients
  },
})
```

For detailed configuration documentation, see `:help cursortab-config`.

**Authentication:**

The plugin looks for the API key in this order:
1. `api_key` config option (if set)
2. Environment variable specified by `api_key_env` (default: `SWEEP_AI_TOKEN`)

## Usage

- **Tab Key**: Navigate to cursor predictions or accept completions
- **Esc Key**: Reject current completions
- The plugin automatically shows jump indicators for predicted cursor positions
- Visual indicators appear for additions, deletions, and completions
- Off-screen jump targets show directional arrows with distance information

### Commands

- `:CursortabToggle`: Toggle the plugin on/off
- `:CursortabShowLog`: Show the cursortab log file in a new buffer
- `:CursortabClearLog`: Clear the cursortab log file
- `:CursortabStatus`: Show detailed status information about the plugin and
  daemon
- `:CursortabRestart`: Restart the cursortab daemon process

## Development

### Build

To build the server component:

```bash
cd server && go build
```

### Test

To run tests:

```bash
cd server && go test ./...
```

## FAQ

<details>
<summary>Why are completions slow?</summary>

1. Decrease `provider.max_tokens` to reduce output length
2. Check your network connection to the Sweep API

</details>

<details>
<summary>Why are completions not working?</summary>

1. Verify your `SWEEP_AI_TOKEN` environment variable is set correctly
2. Update to the latest version and restart the daemon with `:CursortabRestart`
3. Increase `provider.completion_timeout` (default: 5000ms) if experiencing timeouts

</details>

<details>
<summary>How do I update the plugin?</summary>

Use your Neovim plugin manager to pull the latest changes, then run
`:CursortabRestart` to restart the daemon.

</details>

## Contributing

Contributions are welcome! Please open an issue or a pull request.

Feel free to open issues for bugs :)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file
for details.
