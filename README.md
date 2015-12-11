# mackerel-plugin-unicorn-stats
Unicorn metrics plugin for mackerel.io agent.  
Only work on Linux.  
Inspired by [mackerel-plugin-nginx](https://github.com/mackerelio/mackerel-agent-plugins/tree/master/mackerel-plugin-nginx).

## Synopsis

```shell
mackerel-plugin-unicorn-stats [-address=<url or unix domain socket>] [-path=<path>] [-metric-key=<metric-key>]  [-tempfile=<tempfile>]
```

## Requirements

- [raindrops](https://rubygems.org/gems/raindrops)

## Example of mackerel-agent.conf

```
[plugin.metrics.unicorn_stats]
command = "/path/to/mackerel-plugin-unicorn-stats -address=unix:/path/to/unicorn.sock"
```
