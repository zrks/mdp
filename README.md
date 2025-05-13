# intro
Here will be some observations.

## motivation
* learn
* get better

this is human written blog


### where we can see packets

for the last ten years I have been mostly working and learning stuff that comes in handy right away - little have dealt with grasping the whole thing. to me it seems that to truly understand something you must be able to reconstruct it. so I have chosen to go over journey the packet makes when request is sent and received as it imho covers all the stuff you might want to know in order to debug various issues that pops up from time to time in different stages of abstraction.

so there are some tools that Brendan Gregg has mentioned [here](https://github.com/brendangregg/perf-tools).
lets see what can be useful to take a closer look at data at various points.

We have two options how to initiate request we will inspect - via browser or curl - most simple one in terms of traceability would be curl with which we will proceed with an example. imho most obvious choice for checking what is happening under the hood is strace. as I am on mac then from what I read dtrace is a go to tool on bsd systems - [here is a good talk on this tool by charismatic bryan cantrill](https://www.youtube.com/watch?v=TgmA48fILq8). however due to SIP - system integrity protection - it is impossible to run it against curl, which resides under `/usr/bin`, and copying curl binary and doing `codesign --remove-signature ./curl` didn't help me either.