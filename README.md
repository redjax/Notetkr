# Notetkr

`notetkr` is a terminal-based note taking/journaling app.

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

- Be file-based with a configurable backend.
  - Default to files at a specified, standard location, i.e. `~/.notetkr/notes` or `~/.notetkr/journal`.
  - Optionally support other types of backends:
    - SQLite
    - Encrypted files
    - Git
    - S3
- Use sane defaults, with a configuration file to override.
  - Config should live at either `~/.config/notetkr/` or `~/.notetkr`.
  - Accept multiple config filetypes, `.yml` or `.toml` first.
  - Be configurable from the environment.
- Use Markdown for as much as possible, or other open formats where Markdown isn't feasible.
- Offer import/export and backup functionality.

