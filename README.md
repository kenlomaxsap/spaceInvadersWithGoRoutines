# spaceInvadersWithGoRoutines :)

![spaceInvadersWithGoRoutines](https://raw.githubusercontent.com/kenlomaxhybris/spaceInvadersWithGoRoutines/master/spaceInvadersWithGoRoutines.png)

To learn how to work with GoRoutines, Channels, State I implemented a Space Invaders Game :)

  - Usage: 
    - git clone https://github.com/kenlomaxhybris/spaceInvadersWithGoRoutines.git
    - cd spaceInvadersWithGoRoutines/
    - either from binary:
      - ./spaceInvadersWithGoRoutines 
    - or from code..
      - go get
      - go run spaceInvadersWithGoRoutines.go
      - go run spaceInvadersWithGoRoutines.go -flagPieceHeartBeatMS 10 -flagMotherShipHeartBeatMS 10 -flagWidthPx 600 -flagHeightPx 500 -flagSpacingPx 40

# Lessons Learned
  - Do not share state across GoRoutines.
  - Use Selects and Channels wisely
  - If channels are filling up, find a way to empty them more quickly
  
# Helpful resources
- https://gobyexample.com/
