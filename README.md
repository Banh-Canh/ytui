# ytui - YouTube Terminal User Interface

`ytui` is a terminal-based tool designed to help you search and
play YouTube videos directly from your local terminal player.
You can query videos from various sources
such as your history, subscribed channels, or by searching YouTube.

<div style="text-align: center;">
    <img src="public/ytui-demo.gif" width="700vw" />
</div>

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
  - [General Commands](#general-commands)
  - [Query Commands](#query-commands)
- [Configuration](#configuration)
- [Files](#files)
- [Examples](#examples)
- [Troubleshooting](#troubleshooting)
- [License](#license)

## Features

`ytui` offers a wide range of features designed to enhance
your YouTube experience directly from the terminal. Some of the key features include:

- **YouTube Search and Playback**: Search YouTube videos using keywords
  or retrieve videos from your history or subscribed channels,
  and play them directly in your local terminal video player.

- **FZF Integration**: `ytui` features something similar the popular fuzzy finder
  `fzf` (powered by <https://github.com/ktr0731/go-fuzzyfinder>) to allow fast
  and efficient video search results navigation.
  This makes it easy to browse through large lists of videos
  and pick the one you want to play.

- **Video History Management**: Keep track of the videos you've watched using `ytui`.
  The tool logs your watch history in the `watched_history.json`
  file for quick reference later.

- **Channel Subscription Support**: Search for videos from
  your subscribed YouTube channels or specify channels in the configuration file.

- **OAuth Authentication**: Securely authenticate with YouTube
  using OAuth to access your personal subscriptions.
  The tool leverages the YouTube Data API to fetch data related to your account.

- **Terminal-Based User-interface**: Allowing you to search for videos
  directly from the terminal environment.

- **Customization**: Adjust settings through a configuration file
  located in `$HOME/.config/ytui/config.yaml` to tailor the tool
  to your preferences, such as channel subscriptions and OAuth credentials.

These features make `ytui` a powerful and convenient way to interact
with YouTube directly from your terminal, streamlining video searching
and viewing for command-line users.

## Requirements

You must have `mpv` installed.

## Installation

To install `ytui`, follow the instructions for your operating system.
Ensure that you have the required dependencies installed.

## Usage

See [Documentations](docs/ytui.md).

## Configuration

The configuration file for `ytui` is located at `$HOME/.config/ytui/config.yaml`.
This file allows you to specify which channels to subscribe to and your OAuth credentials.

### Configuration File Structure

```yaml
channels:
  local: false
  subscribed:
    - UCTt2AnK--mnRmICnf-CCcrw
    - UCutXfzLC5wrV3SInT_tdY0w
invidious:
  instance: invidious.jing.rocks
loglevel: info
youtube:
  clientid: fsdfsdf
  secretid: ffsdfsdf
```

#### Notes

- **`local: false`** - When set to `true`, `ytui` will use the channels
  specified in the configuration file for subscribed channels.
  If you prefer to use your Youtube user-subscribed channels, set this to `false`.

- `channels.subscribed: []` is a list of channel Ids. To be used with `local: true`.

- **OAuth** - You need to enable OAuth authentication with YouTube
  to access your subscribed channels.
  Ensure that your `clientid` and `secretid` are properly configured.

  The following scope is also required: `https://www.googleapis.com/auth/youtube.readonly`

## Files

- **`watched_history.json`** - This file, located in `$HOME/.config/ytui/`,
  logs each video watched using `ytui` when querying the history.

## Examples

1. **Search for a Video on YouTube:**

   ```sh
   ytui query search "your search query"
   ```

1. **Fetch Videos from Your Subscribed Channels:**

   ```sh
   ytui query subscribed
   ```

1. **Retrieve Videos from Your History:**

   ```sh
   ytui query history
   ```

## Troubleshooting

If you encounter any issues while using this application, you can check the log file for detailed error messages and troubleshooting information. The log file is located at:

```
$HOME/.config/ytui/ytui.log
```

### Steps to View the Log

1. **Open a terminal**: Use your terminal application to navigate to the logs.

2. **Display the log**: Run the following command to view the log content:

   ```bash
   cat $HOME/.config/ytui/ytui.log
   ```

   This will print the log file's contents directly to your terminal.

3. **Tail the log** _(Optional)_: If the application is still running and you want to monitor log updates in real-time, you can use the `tail` command:

   ```bash
   tail -f $HOME/.config/ytui/ytui.log
   ```

4. **Check for errors or warnings**: Look for lines marked with `[ERROR]` or `[WARNING]` to identify issues.

### Reporting Issues

When reporting issues, please include relevant log entries to help with diagnosing the problem. You can copy the log content and include it in your bug report.

## License

`ytui` is open-source and available under the [MIT License](LICENSE).

For more detailed usage, you can always use `ytui --help` or `ytui <subcommand> --help`
to get more information about specific commands and flags.
