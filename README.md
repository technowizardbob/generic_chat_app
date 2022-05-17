# Generic LAN CHAT 
My very first attempts at a generic WINPOPUP 'ish like client for Linux...

## Made in Go.

## Server seems okay...but the Client needs work.

Lots of bugs: 


- [x] 1) Lanchat crashes at times with a segfault.
- [x] 2) Needs to pop-up on new message.
- [ ] 3) Does not list old messages. 
- [-] 4) Will only show/exchange messages to currently connected users....
- [x] 5) Needs a Scroll Window, so TreeView Chat stays inside of the Window!
- [x] 6) Client does not reconnect gracefully.
- [x] 7) Needs to auto scroll down on new message.

### Be sure to update the conf.yaml files to allow access...

To modify the GUI have Glade installed:
```
$ sudo apt update
$ sudo apt -y install glade
```

[See what it looks like, yikes!](LanChat-Client.png)
