# Notetkr <!-- omit in toc -->

<!-- Badges/shields -->
<p align="center">
  <img alt="GitHub Created At" src="https://img.shields.io/github/created-at/redjax/Notetkr">
  <img alt="GitHub Last Commit" src="https://img.shields.io/github/last-commit/redjax/Notetkr">
  <img alt="GitHub Commits this Year" src="https://img.shields.io/github/commit-activity/y/redjax/Notetkr">
  <img alt="Github Repo Size" src="https://img.shields.io/github/repo-size/redjax/Notetkr">
</p>

`notetkr` is a terminal-based note taking/journaling app. It is cross-platform and implemented in pure Go for compatibility. The editor UI is a modal editor similar to/inspired by Neovim.

> [!WARNING]
> This is a personal project to satisfy a personal need in my working life. I wanted a quick, cross-platform, easy to use app for jotting down daily tasks and generating a weekly summary document at the end of the week for reporting.
>
> The app was built to my own personal specifications and may not be useful outside of my own personal use case.

## Table of Contents <!-- omit in toc -->

- [Install](#install)
  - [Install script](#install-script)
  - [From Release](#from-release)
  - [Build locally](#build-locally)
- [Usage](#usage)
  - [Editing](#editing)
- [Purpose](#purpose)

## Install

### Install script

You can download & install Notetkr on Linux & Mac using [the `install-notetkr.sh` script](./scripts/install-notetkr.sh).

To download & install in 1 command, do:

```bash
curl -LsSf https://raw.githubusercontent.com/redjax/Notetkr/refs/heads/main/scripts/install-notetkr.sh | bash -s -- --auto
```

For Windows, use:

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/redjax/Notetkr/refs/heads/main/scripts/install-notetkr.ps1))) -Auto
```

### From Release

Install a release from the [releases page](https://github.com/redjax/Notetkr/releases/latest).

### Build locally

- Clone the repository with `git clone https://github.com/redjax/Notetkr`.
- `cd` into the newly cloned directory.
- Build using Go or Goreleaser:
  - Build with Go:
    - `go build -o ./dist/nt.exe -ldflags "-X 'main.buildType=development'" ./cmd/entrypoints/main.go`
  - Build with Goreleaser:
    - `goreleaser build --single-target --clean --snapshot -o dist/nt.exe`
- Build using a script.
  - The [`build-dev.ps1` script](./scripts/build-dev.ps1) builds the app locally (with Go by default, but has a `-UseGoReleaser` flag).
- After building, launch with `./dist/nt.exe [OPTIONS]`, or put the binary somewhere in your `$PATH` to make it available globally.

## Usage

Start the terminal UI (TUI) with `nt`. See the usage menu with `nt --help`. You can also open journals or notes directly with:

```shell
## Open straight to journals UI
nt journal

## Open straight to today's journal entry
nt journal today

## Open straight to notes UI
nt notes
```

You can export Notetkr's data, and later re-import it, with `nt export` and `nt import`:

```shell
## Export data to ~/<YYYY-mm-dd>-notetkr.zip
nt export

## Export only notes to a specific path
nt export -o ~/Downloads/notetkr-export.zip -t notes

## Import data
nt import -f ~/Downloads/notetkr-export.zip
```

Notetakr stores its files in `$HOME/.notetkr`. Journals & summaries are in `$HOME/.notetkr/journal` and notes are stored in `$HOME/.notetkr/notes`.

To generate a new weekly summary, run `nt journal` and press the `g` key to open the "generate summary" menu.

### Editing

When opening a journal entry or note, you will be presented with a modal editor similar to Neovim. Navigate around with `hjkl` or the arrow keys, and switch to INSERT mode with `i`. You can press `g` to put the cursor at the top of the document, or `G` to go to the bottom. Pressing `d` will delete the current line the cursor is on. `CTRL+Z` undoes a change and `CTRL+Y` redoes it.

There are keypress hints along the bottom of the editor to help remember these shortcuts.

## Purpose

I've used various tools to keep track of things I do throughout the day, specifically at work, and thought...why not write my own tool?

2 tools I've used that I've really enjoyed using are Neovim plugins:

- [Neorg](https://github.com/nvim-neorg/neorg) - Inspired by Emacs' ORG-mode, provides a ton of tools and utilities for organizing your life in the terminal.
  - Liked:
    - Interface intuitive keybinds & snappy buffer switching.
    - File-based backend, all my notes in `~/.norg`
    - Markdown-like syntax
    - Portability
  - Disliked:
    - Brittle...updating Neovim caused it to break constantly.
    - Had to write custom functionality for merging daily reports into weekly.
- [journal.nvim](https://github.com/jakobkhansen/journal.nvim) - Extensible note-taking system in Neovim.
  - Liked:
    - Simple, small starting point that you can build on top of.
    - File-based backend, all my notes in a configurable location (I picked `~/.journal`.
    - Snappy.
    - Fully Markdown based
  - Disliked:
    - Not much, so far!

`notetkr` should:

- [ ] Be file-based with a configurable backend.
  - [x] Default to files at a specified, standard location, i.e. `~/.notetkr/notes` or `~/.notetkr/journal`.
  - [ ] Optionally support other types of backends:
    - [ ] SQLite
    - [ ] Encrypted files
    - [ ] Git
    - [ ] S3
- [ ] Use sane defaults, with a configuration file to override.
  - [ ] Config should live at either `~/.config/notetkr/` or `~/.notetkr`.
  - [ ] Accept multiple config filetypes, `.yml` or `.toml` first.
  - [ ] Be configurable from the environment.
- [x] Use Markdown for as much as possible, or other open formats where Markdown isn't feasible.
- [x] Offer import/export and backup functionality.
