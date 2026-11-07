// Trisha v1: shared reconnaissance, directed flag search and mass production.
// Only documented sensors are used; no map or opponent-code access.
// One action per call. Small bounded loops keep interpreter work low.
type Memory struct {
	tx, ty, budget, moves, scan, dodge, steps int
	visits map[int]int
   }
   var team = map[int]*Memory{}
   var walls = map[int]bool{}
   var clear = map[int]bool{}
   var blocked = map[int]int{}
   var tick, lastID int
   var flagKnown bool
   var flagX, flagY int
   var dx = []int{0,1,0,-1}
   var dy = []int{-1,0,1,0}
   func abs(x int) int { if x < 0 { return -x }; return x }
   func key(x,y int) int { return x*1000+y }
   func face(d int) bool {
	if d == get_dir() { return true }
	if (d-get_dir()+4)%4 == 3 { turn_left() } else { turn_right() }
	return false
   }
   func observe(what string, dist int) {
	x,y := get_pos(); d := get_dir(); g := get_grid_size()
	clear[key(x,y)] = true
	for i:=1; i<dist; i++ { clear[key(x+dx[d]*i,y+dy[d]*i)] = true }
	ex,ey := x+dx[d]*dist,y+dy[d]*dist
	if ex<0 || ey<0 || ex>=g || ey>=g { return }
	k:=key(ex,ey)
	if what=="wall" || what=="my_flag" { walls[k]=true; clear[k]=true }
	if what=="ally" || what=="construction" { blocked[k]=tick+3 }
	if what=="flag" || what=="enemy_flag" { flagKnown=true; flagX=ex; flagY=ey }
   }
   func navigate(u *Memory) {
	x,y:=get_pos(); g:=get_grid_size(); dir:=get_dir()
	best,want:=1000000,dir
	for i:=0;i<4;i++ {
	 d:=(dir+i)%4; nx,ny:=x+dx[d],y+dy[d]; k:=key(nx,ny)
	 if nx<0 || ny<0 || nx>=g || ny>=g || walls[k] { continue }
	 score:=4*(abs(nx-u.tx)+abs(ny-u.ty))+u.visits[k]*9
	 turn:=(d-dir+4)%4; if turn==3 {turn=1}; score+=turn
	 if blocked[k]>tick {score+=50}
	 if score<best {best=score;want=d}
	}
	if face(want) {
	 what,dist:=look_forward()
	 if dist==1 && what=="built_wall" && my_kind()!="builder" {shoot();return}
	 if dist==1 && (what=="ally" || what=="construction" || what=="bullet") {
	  blocked[key(x+dx[want],y+dy[want])]=tick+4
	  u.visits[key(x,y)]++;turn_right();return
	 }
	 if move_forward() {u.moves++;u.steps++;nx,ny:=get_pos();u.visits[key(nx,ny)]++} else {
	  blocked[key(x+dx[want],y+dy[want])]=tick+5;u.visits[key(x,y)]++
	 }
	}
	u.budget--
   }
   func target(u *Memory,x,y int) {
	u.tx=x;u.ty=y;u.budget=120;u.visits=map[int]int{}
   }
   func search(u *Memory) {
	x,y:=get_pos();g:=get_grid_size()
	if flagKnown {if u.tx!=flagX || u.ty!=flagY {target(u,flagX,flagY)};navigate(u);return}
	lo,hi:=g/2-5,g/2+4
	if lo<0 {lo=0};if hi>=g {hi=g-1}
	if game_mode()=="ctf" {
	 home:=g/4;if home>8 {home=8};fx,_:=my_flag()
	 lo,hi=g-2-home,g-3;if fx>=g/2 {lo,hi=2,1+home}
	}
	if u.budget<=0 || clear[key(u.tx,u.ty)] || (x==u.tx && y==u.ty) {
	 best,bx,by:=1000000,lo,lo
	 for a:=lo;a<=hi;a++ {for b:=lo;b<=hi;b++ {
	  k:=key(a,b);if clear[k] || walls[k] {continue}
	  score:=abs(x-a)+abs(y-b)
	  // Separate warriors rather than queue behind the same target.
	  if game_mode()=="ctf" {score+=((a+b+my_id())%3)*2}
	  if score<best {best=score;bx=a;by=b}
	 }}
	 if best==1000000 {bx=lo+randInt(hi-lo+1);by=lo+randInt(hi-lo+1)}
	 target(u,bx,by)
	}
	// In classic, reach the flag region before spending turns scanning.
	if u.moves>=6 && (game_mode()!="classic" || (x>=lo && x<=hi && y>=lo && y<=hi)) {u.moves=0;u.scan=3;turn_right();return}
	navigate(u)
   }
   func loop() {
	id:=my_id();if id<=lastID {tick++};lastID=id
	u,ok:=team[id];if !ok {u=&Memory{visits:map[int]int{}};team[id]=u}
	what,dist:=look_forward();observe(what,dist)
	if busy()>0 {return}
	if u.dodge>0 {
	 u.dodge=0
	 if dist>1 || what=="flag" || what=="enemy_flag" {move_forward()} else {turn_right()}
	 return
	}
	if what=="bullet" && dist<=8 {
	 x,y:=get_pos();d:=(get_dir()+1)%4;g:=get_grid_size()
	 if x+dx[d]<0 || y+dy[d]<0 || x+dx[d]>=g || y+dy[d]>=g || walls[key(x+dx[d],y+dy[d])] {turn_left()} else {turn_right()}
	 u.dodge=1;return
	}
	if what=="enemy" && my_kind()!="builder" {shoot();return}
	if what=="flag" || what=="enemy_flag" {u.scan=0;move_forward();return}
	if my_kind()=="builder" {
	 if my_units()<6 {
	  if !build_warrior() {turn_right()};return
	 }
	 // Preserve the production base; keep an escape/production lane open.
	 turn_right();return
	}
	if u.scan>0 {u.scan--;turn_right();return}
	if game_mode()=="deathmatch" {
	 // Defend while the opponent travels; hunt only late to resolve ties.
	 if tick<1400 {turn_right();return}
	 x,y:=get_pos();g:=get_grid_size()
	 if u.budget<=0 || (x==u.tx && y==u.ty) {target(u,2+randInt(g-4),2+randInt(g-4))}
	 if u.moves>=4 {u.moves=0;u.scan=3;turn_right();return}
	 navigate(u);return
	}
	search(u)
   }
   