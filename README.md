<p align="center">
  <h1>🤖cy</h1>
</p>

> the time traveling terminal multiplexer

<p align="center">
    <img src="gh-assets/screenshot.png" alt="Cy Cover Image">
</p>

<p align="center">
    <img src="https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white" alt="Go Language" />
    <!-- LICENSE -->
    <a target="_blank" href="https://github.com/cfoust/cy/blob/main/LICENSE">
        <img src="https://img.shields.io/github/license/cfoust/cy" alt="cy License Badge MIT" />
    </a>
</p>

## What is this?

`cy` is an experimental terminal multiplexer that aims to be a simple, modern, and ergonomic alternative to `tmux`.

I have used `tmux` for nearly a decade and while it is a powerful tool, it has its fair share of inadequacies. I built a plugin, [tmux-oakthree](https://github.com/cfoust/tmux-oakthree) that simplifies tmux to match my workflow exactly, but over time I found that I still wanted more.

The `cy` project has a few main goals:
* Be beautiful, fast, and easy-to-use, particularly for users who have not used `tmux` or who are intimidated by its default configuration.
* Allow users to search in and replay any `cy` session.
* Stay simple. `cy` is specifically _not_ a clone of `tmux` and will never have all of its functionality (panes, etc).

## Roadmap

* [ ] **Usability**
* [ ] **Development**
    * [ ] Implement [oakthree](https://github.com/cfoust/tmux-oakthree)
        * [ ] Allow users to kill parts of the tree
        * [ ] Store pane jump history
    * [ ] Allow for providing binding scope
    * [ ] Logging API in Janet
        * [ ] Support unmarshaling Janet table to `map[string]interface{}`
* [ ] **Replay**
    * [ ] Record all sessions and save in custom file format
        * [ ] Is this also the `.borg` file format?
    * [ ] Allow users to search through history
    * [ ] Don't record bytes matching regex (IPs etc)
