An old solver I wrote in Go for the game Rummikub, along with a little game server based on WebSockets. At the time, the goal was to get a feel for the Go language and to experiment with different discrete optimization methods. Oh, yes, and also to prove that I am actually very good at playing Rummikub, just not when I have to play it in person.

# Rummikub
Not everyone plays Rummikub the same way. This program maps the search space of possible tile combinations according to a set of rules that may deviate from other game styles. Some of the ways in which my implementation of the game of Rummikub may differ:

- A group or a run of tiles may only contain a single Joker

- Initial move must have a value of at least 14

  

# TODO

- [ ] see all `TODO` tags in the code
- [ ] clean up all 'prototypey' code
- [ ] clean up the math
- [ ] build web UI
- [ ] finish tests for the websockets game server
- [ ] put all the server code in a docker container, because `libglpk` does not behave well outside of Ubuntu.
- [ ] cite paper that inspired the ILP formulation of the Rummikub problem
- [ ] rephrase 'set' and 'row' into 'group' and 'run' to prevent confusion

# Dependencies

Tested on **Go 1.10**.

Uses **libglpk** as the solver for the mixed integer linear program.

```
sudo apt-get install libglpk-dev
```

Or, on macOs:

```
brew install glpk
```

