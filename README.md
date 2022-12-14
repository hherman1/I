# I

I is a command for making cli tools interactive in the Acme editor. 

Execute `I <cli>` and your command will be executed in the body of a new Acme window. 

* Button 2 clicks on text in the window will append the clicked text as a new argument for your command, clear the output, and re-execute it. 
* Button 2 of the `Back` command in the tag will remove the newest argument and rerun. 
* The `Get` command rerun's the command as is.

# Demo

Here's a very simple demo of me clicking around the `go` program:

https://user-images.githubusercontent.com/611822/187044748-6e4fda56-3e91-4692-8709-39f1229d812f.mp4

In the video I run `I go` then use button 2 clicks to navigate the subcommands and execute some of them. 

# Install

First, make sure you've installed [the Acme editor](https://github.com/9fans/plan9port), then run `go install github.com/hherman1/I@latest`. 

Thats it! Now you can run `I <cli>` anywhere to launch an I session in Acme.  
