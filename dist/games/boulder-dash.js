// boulder-dash.js — Boulder Dash for dev-null
// Load with: /game load boulder-dash

// ============================================================
// Tile constants
// ============================================================
var EMPTY=0, DIRT=1, WALL=2, BOULDER=3, DIAMOND=4,
    EXIT_C=5, EXIT_O=6, AMOEBA=7, MAGIC_WALL=8;

// Directions
var UP=0, DOWN=1, LEFT=2, RIGHT=3;
var DX=[0,0,-1,1], DY=[-1,1,0,0];
// TURN_L[d] = direction 90° to the left of d
// UP→LEFT, DOWN→RIGHT, LEFT→DOWN, RIGHT→UP
var TURN_L=[2,3,1,0];
// TURN_R[d] = direction 90° to the right of d
// UP→RIGHT, DOWN→LEFT, LEFT→UP, RIGHT→DOWN
var TURN_R=[3,2,0,1];
// OPP[d] = opposite of d
var OPP=[1,0,3,2];

// ============================================================
// Timing & scoring constants
// ============================================================
var PHYSICS_INTERVAL = 0.15;
var ENEMY_INTERVAL   = 0.30;
var AMOEBA_INTERVAL  = 0.40;
var RESPAWN_TIME     = 3.0;
var INVULN_TIME      = 2.0;
var CAVE_WIN_DELAY   = 2.5;
var AMOEBA_MAX       = 200;
var PTS_DIAMOND      = 10;
var PTS_TIME         = 5;
var PTS_CAVE         = 500;

// ============================================================
// Colors
// ============================================================
var C_DIRT_FG="#AA5500", C_DIRT_BG="#331100";
var C_WALL_FG="#888888", C_WALL_BG="#444444";
var C_BOULDER="#AAAAAA";
var C_DIA_A="#00FFFF", C_DIA_B="#005588";
var C_FIREFLY="#FF4400";
var C_BUTTERFLY="#FF44FF";
var C_AMOEBA="#00AA00";
var C_MAGIC_FG="#AA00AA", C_MAGIC_BG="#220022";
var C_EXIT_C="#555555";
var C_EXIT_O="#00FF00";
var C_PLAYER="#FFFF00";
var C_OTHER="#FF8800";
var C_DEAD_FG="#444444";
var C_EXPLO="#FF8800";

// ============================================================
// Cave definitions
// Cave chars: #=wall .=dirt ' '=empty O=boulder *=diamond
//             P=player-start X=exit Q=firefly B=butterfly
//             A=amoeba M=magic-wall
// Each row is padded to the max-row-length with '#' by parseGrid.
// ============================================================
var CAVES = [
  // ── Cave A ─────────────────────────────────────────────────
  {
    name: "A", title: "Rookie Mine",
    diamondsNeeded: 6, timeLimit: 150, magicWallDur: 0,
    raw: [
      "########################################",
      "#......................................P#",
      "#....*.......O.........................#",
      "#......................................#",
      "#.O....*.............O.................#",
      "#......................................#",
      "#...............*......................#",
      "#......................................#",
      "#....*.O.......O.......................#",
      "#......................................#",
      "#.......*..............................#",
      "#......................................#",
      "#.O.............O......*...............#",
      "#......................................#",
      "#.......*..............................#",
      "#......................................#",
      "#.X....................................#",
      "########################################"
    ]
  },
  // ── Cave B ─────────────────────────────────────────────────
  {
    name: "B", title: "Rolling Stones",
    diamondsNeeded: 10, timeLimit: 120, magicWallDur: 0,
    raw: [
      "########################################",
      "#P..*..OOO.......OO...*................#",
      "#......................................#",
      "#..*.....O.O...*......OOOO.............#",
      "#......................................#",
      "#.OOOO.........*.....O.O.....*........#",
      "#......................................#",
      "##########                  ###########",
      "#Q        .....*............Q         #",
      "#         .....*............          #",
      "##########                  ###########",
      "#.*....OO..........OO.*.................#",
      "#......................................#",
      "#.......*....OOOO....*........OO......#",
      "#......................................#",
      "#.OO.....*............................#",
      "#.....................................X#",
      "########################################"
    ]
  },
  // ── Cave C ─────────────────────────────────────────────────
  // Butterflies turn into 3×3 diamonds when crushed.
  // Dig under the boulders above their rooms to release them.
  {
    name: "C", title: "Butterfly Garden",
    diamondsNeeded: 12, timeLimit: 120, magicWallDur: 0,
    raw: [
      "########################################",
      "#P......................................#",
      "#.......OOO.....*.......OOO............#",
      "#......................................#",
      "##########.##########.##########.######",
      "#B                   B                #",
      "#         ...........                 #",
      "#         ...........                 #",
      "##########.##########.##########.######",
      "#......................................#",
      "#......OOO......*.......OOO............#",
      "#......................................#",
      "##########.##########.##########.######",
      "#B                   B                #",
      "#         ...........                 #",
      "#         ...........                 #",
      "##########.##########.##########.######",
      "#......*.....*......*.....*..........X#",
      "########################################"
    ]
  },
  // ── Cave D ─────────────────────────────────────────────────
  {
    name: "D", title: "Firefly Alley",
    diamondsNeeded: 14, timeLimit: 100, magicWallDur: 0,
    raw: [
      "########################################",
      "#P...*......*.......*.....*............#",
      "#.###########.###########.###########.#",
      "#.#Q         #Q          #Q          .#",
      "#.#          #           #           .#",
      "#.#          #           #           .#",
      "#.#          #           #           .#",
      "#.###########.###########.###########.#",
      "#......................................#",
      "#...*......*.......*......*............#",
      "#.###########.###########.###########.#",
      "#.#Q         #Q          #           .#",
      "#.#          #           #           .#",
      "#.#          #           #           .#",
      "#.#          #           #           .#",
      "#.###########.###########.###########.#",
      "#...*......*.......*......*.............#",
      "#.....................................X#",
      "########################################"
    ]
  },
  // ── Cave E ─────────────────────────────────────────────────
  // Features: magic walls (boulders→diamonds when they pass through),
  // amoeba (grows, suffocate it for diamonds), both enemy types.
  {
    name: "E", title: "The Gauntlet",
    diamondsNeeded: 20, timeLimit: 90, magicWallDur: 25,
    raw: [
      "########################################",
      "#P....*......*......*......*..........#",
      "#......................................#",
      "#..OOOOOO...OOOOOO...OOOOOO...OOOOOO.#",
      "#......................................#",
      "#MMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMM#",
      "#......................................#",
      "#....AA......AA......AA......AA.......#",
      "#....AA......AA......AA......AA.......#",
      "#......................................#",
      "#################  ####################",
      "#Q               ##B                  #",
      "#                ##                   #",
      "#                ##                   #",
      "#################  ####################",
      "#......................................#",
      "#..*.......*.....*.......*...........#",
      "#.....................................X#",
      "########################################"
    ]
  }
];

// ============================================================
// Module-level state (all reset in loadCave / begin)
// ============================================================
var grid=[], fallingGrid=[], movedGrid=[];
var CAVE_W=0, CAVE_H=0;
var enemies=[];     // [{type:"firefly"|"butterfly", x, y, dir}]
var amoebaList=[];  // [{x,y}] current amoeba cells
var pls={};         // per-player records
var plOrder=[];     // insertion-order player IDs

var caveIndex=0, caveData=null;
var caveName="", caveTitle="";
var startX=1, startY=1;
var diamondsNeeded=0, diamondsCollected=0;
var timeLeft=0;
var exitOpen=false;
var magicWallActive=false, magicWallTimer=0;
var physicsTimer=0, enemyTimer=0, amoebaTimer=0;
var elapsed=0;
var wonDelay=0;     // > 0 while waiting for next-cave transition
var gameEnded=false;
var highScore=0;

// ============================================================
// Helpers
// ============================================================
function clamp(v,lo,hi){ return v<lo?lo:v>hi?hi:v; }
function rng(){ return Math.random(); }

function inBounds(x,y){
    return x>=0 && x<CAVE_W && y>=0 && y<CAVE_H;
}

function playerAt(x,y){
    for(var id in pls){
        var p=pls[id];
        if(!p.dead && !p.exited && p.x===x && p.y===y) return id;
    }
    return null;
}

function enemyAt(x,y){
    for(var i=0;i<enemies.length;i++){
        if(enemies[i].x===x && enemies[i].y===y) return i;
    }
    return -1;
}

// ============================================================
// Cave loading
// ============================================================
function parseGrid(raw){
    var maxW=0;
    for(var i=0;i<raw.length;i++){
        if(raw[i].length>maxW) maxW=raw[i].length;
    }
    CAVE_H=raw.length;
    CAVE_W=maxW;
    var g=[], fg=[], mg=[], ens=[], amL=[], sx=1, sy=1;
    for(var y=0;y<CAVE_H;y++){
        var row=raw[y];
        while(row.length<maxW) row+="#";
        g.push([]); fg.push([]); mg.push([]);
        for(var x=0;x<maxW;x++){
            var ch=row[x], tile=EMPTY;
            switch(ch){
                case '#': tile=WALL; break;
                case '.': tile=DIRT; break;
                case 'O': tile=BOULDER; break;
                case '*': tile=DIAMOND; break;
                case 'X': tile=EXIT_C; break;
                case 'A': tile=AMOEBA; amL.push({x:x,y:y}); break;
                case 'M': tile=MAGIC_WALL; break;
                case 'P': tile=EMPTY; sx=x; sy=y; break;
                case 'Q': tile=EMPTY; ens.push({type:"firefly",x:x,y:y,dir:RIGHT}); break;
                case 'B': tile=EMPTY; ens.push({type:"butterfly",x:x,y:y,dir:LEFT}); break;
                default:  tile=EMPTY; break;  // space and unknown = empty
            }
            g[y].push(tile); fg[y].push(false); mg[y].push(false);
        }
    }
    return {g:g,fg:fg,mg:mg,ens:ens,amL:amL,sx:sx,sy:sy};
}

function loadCave(idx){
    caveData          = CAVES[idx];
    caveIndex         = idx;
    caveName          = caveData.name;
    caveTitle         = caveData.title;
    diamondsNeeded    = caveData.diamondsNeeded;
    diamondsCollected = 0;
    timeLeft          = caveData.timeLimit;
    exitOpen          = false;
    magicWallActive   = (caveData.magicWallDur > 0);
    magicWallTimer    = caveData.magicWallDur || 0;
    wonDelay          = 0;
    physicsTimer=0; enemyTimer=0; amoebaTimer=0;

    var parsed=parseGrid(caveData.raw);
    grid=parsed.g; fallingGrid=parsed.fg; movedGrid=parsed.mg;
    enemies=parsed.ens; amoebaList=parsed.amL;
    startX=parsed.sx; startY=parsed.sy;

    // Place / reset all living players
    for(var id in pls){
        var p=pls[id];
        if(p.lives>0){
            p.x=startX; p.y=startY;
            p.dead=false; p.exited=false;
            p.respawnTimer=0; p.invulnTimer=INVULN_TIME;
        }
    }
}

// ============================================================
// Physics
// ============================================================
function clearMoved(){
    for(var y=0;y<CAVE_H;y++)
        for(var x=0;x<CAVE_W;x++)
            movedGrid[y][x]=false;
}

// Called when a heavy object lands at (x,y).
function checkCrush(x,y){
    var pid=playerAt(x,y);
    if(pid!==null) triggerDeath(pid);

    var ei=enemyAt(x,y);
    if(ei>=0){
        var yTile=(enemies[ei].type==="butterfly")?DIAMOND:EMPTY;
        enemies.splice(ei,1);
        doExplosion(x,y,yTile);
    }
}

function doExplosion(cx,cy,yTile){
    for(var dy=-1;dy<=1;dy++){
        for(var dx=-1;dx<=1;dx++){
            var ex=cx+dx, ey=cy+dy;
            if(!inBounds(ex,ey)) continue;
            if(grid[ey][ex]===WALL) continue;
            grid[ey][ex]=yTile;
            fallingGrid[ey][ex]=false;
            movedGrid[ey][ex]=true;
            // Kill players in blast
            var pid=playerAt(ex,ey);
            if(pid!==null) triggerDeath(pid);
            // Remove any other enemies in blast
            for(var i=enemies.length-1;i>=0;i--){
                if(enemies[i].x===ex && enemies[i].y===ey) enemies.splice(i,1);
            }
        }
    }
    // Diamonds placed by explosions appear in the grid; players collect by walking over them.
}

function physicsTick(){
    clearMoved();
    for(var y=0;y<CAVE_H-1;y++){
        for(var x=0;x<CAVE_W;x++){
            if(movedGrid[y][x]) continue;
            var tile=grid[y][x];
            if(tile!==BOULDER && tile!==DIAMOND) continue;

            var below=grid[y+1][x];

            // ── Magic wall pass-through ──────────────────────
            if(below===MAGIC_WALL && magicWallActive){
                if(y+2<CAVE_H && grid[y+2][x]===EMPTY && !movedGrid[y+2][x]){
                    grid[y][x]=EMPTY;
                    fallingGrid[y][x]=false;
                    var out=(tile===BOULDER)?DIAMOND:BOULDER;
                    grid[y+2][x]=out;
                    fallingGrid[y+2][x]=true;
                    movedGrid[y+2][x]=true;
                    checkCrush(x,y+2);
                }
                continue;
            }

            // ── Direct fall ──────────────────────────────────
            // Note: no player/enemy exclusion — falling onto them is the crush mechanic.
            if(below===EMPTY && !movedGrid[y+1][x]){
                grid[y][x]=EMPTY;
                fallingGrid[y][x]=false;
                grid[y+1][x]=tile;
                fallingGrid[y+1][x]=true;
                movedGrid[y+1][x]=true;
                checkCrush(x,y+1);
                continue;
            }

            // ── Roll off round surfaces ──────────────────────
            // Only rolls if was actively falling last tick (fallingGrid).
            if(fallingGrid[y][x] && (below===BOULDER||below===DIAMOND||below===WALL||below===MAGIC_WALL)){
                // Try left
                if(x>0 && grid[y][x-1]===EMPTY && grid[y+1][x-1]===EMPTY && !movedGrid[y][x-1]){
                    grid[y][x]=EMPTY;
                    fallingGrid[y][x]=false;
                    grid[y][x-1]=tile;
                    fallingGrid[y][x-1]=true;
                    movedGrid[y][x-1]=true;
                    continue;
                }
                // Try right
                if(x+1<CAVE_W && grid[y][x+1]===EMPTY && grid[y+1][x+1]===EMPTY && !movedGrid[y][x+1]){
                    grid[y][x]=EMPTY;
                    fallingGrid[y][x]=false;
                    grid[y][x+1]=tile;
                    fallingGrid[y][x+1]=true;
                    movedGrid[y][x+1]=true;
                    continue;
                }
            }

            // Settled this tick
            fallingGrid[y][x]=false;
        }
    }
}

// ============================================================
// Enemy AI
// ============================================================
function moveEnemy(e){
    var prefL=(e.type==="firefly");
    var d0=prefL?TURN_L[e.dir]:TURN_R[e.dir]; // preferred turn
    var d1=e.dir;                               // straight
    var d2=prefL?TURN_R[e.dir]:TURN_L[e.dir]; // opposite turn
    var d3=OPP[e.dir];                          // reverse

    var dirs=[d0,d1,d2,d3];
    for(var i=0;i<4;i++){
        var d=dirs[i];
        var nx=e.x+DX[d], ny=e.y+DY[d];
        if(!inBounds(nx,ny)) continue;
        if(grid[ny][nx]!==EMPTY) continue;
        if(enemyAt(nx,ny)>=0) continue;
        e.dir=d; e.x=nx; e.y=ny;
        return;
    }
}

function enemyTick(){
    for(var i=0;i<enemies.length;i++){
        moveEnemy(enemies[i]);
        var pid=playerAt(enemies[i].x,enemies[i].y);
        if(pid!==null) triggerDeath(pid);
    }
}

function amoebaGrow(){
    var grew=false;
    var snapshot=amoebaList.slice(); // avoid index drift while pushing
    for(var i=0;i<snapshot.length;i++){
        var a=snapshot[i];
        if(grid[a.y][a.x]!==AMOEBA) continue;
        var d=Math.floor(rng()*4);
        var nx=a.x+DX[d], ny=a.y+DY[d];
        if(!inBounds(nx,ny)) continue;
        var t=grid[ny][nx];
        if(t===EMPTY||t===DIRT){
            grid[ny][nx]=AMOEBA;
            amoebaList.push({x:nx,y:ny});
            grew=true;
        }
    }
    // Rebuild authoritative list
    var newL=[];
    for(var y=0;y<CAVE_H;y++)
        for(var x=0;x<CAVE_W;x++)
            if(grid[y][x]===AMOEBA) newL.push({x:x,y:y});
    amoebaList=newL;

    if(amoebaList.length>AMOEBA_MAX){
        for(var ka=0;ka<amoebaList.length;ka++) grid[amoebaList[ka].y][amoebaList[ka].x]=BOULDER;
        amoebaList=[];
        chat("Amoeba too large - turned to boulders!");
    } else if(!grew && amoebaList.length>0){
        for(var kb=0;kb<amoebaList.length;kb++) grid[amoebaList[kb].y][amoebaList[kb].x]=DIAMOND;
        chat("Amoeba suffocated - turned to diamonds!");
        amoebaList=[];
    }
}

// ============================================================
// Player actions
// ============================================================
function openExit(){
    exitOpen=true;
    for(var y=0;y<CAVE_H;y++)
        for(var x=0;x<CAVE_W;x++)
            if(grid[y][x]===EXIT_C) grid[y][x]=EXIT_O;
    chat("Exit open! Get to the X!");
}

function triggerDeath(playerID){
    var p=pls[playerID];
    if(!p||p.dead||p.invulnTimer>0) return;
    p.dead=true;
    p.lives--;
    if(p.lives>0){
        p.respawnTimer=RESPAWN_TIME;
        chatPlayer(playerID,"Crushed! Respawning in "+Math.ceil(RESPAWN_TIME)+"s ("+p.lives+" lives left)");
    } else {
        chatPlayer(playerID,"No lives left! You are now spectating.");
    }
}

function tryMove(playerID, dx, dy){
    var p=pls[playerID];
    if(!p||p.dead||p.exited||p.invulnTimer>0||wonDelay>0) return;
    var tx=p.x+dx, ty=p.y+dy;
    if(!inBounds(tx,ty)) return;
    var t=grid[ty][tx];

    if(t===EMPTY){
        p.x=tx; p.y=ty;
    } else if(t===DIRT){
        grid[ty][tx]=EMPTY;
        p.x=tx; p.y=ty;
    } else if(t===DIAMOND){
        grid[ty][tx]=EMPTY;
        p.x=tx; p.y=ty;
        p.diamonds++;
        p.score+=PTS_DIAMOND;
        diamondsCollected++;
        if(diamondsCollected>=diamondsNeeded && !exitOpen) openExit();
    } else if(t===EXIT_O){
        p.x=tx; p.y=ty;
        p.exited=true;
        p.score+=Math.floor(timeLeft)*PTS_TIME+PTS_CAVE;
        checkCaveWon();
        return;
    } else if(t===BOULDER && dy===0){
        // Horizontal push: one empty space needed behind boulder
        var bx=tx+dx;
        if(inBounds(bx,ty) && grid[ty][bx]===EMPTY && playerAt(bx,ty)===null && enemyAt(bx,ty)<0){
            grid[ty][tx]=EMPTY;
            grid[ty][bx]=BOULDER;
            fallingGrid[ty][bx]=false;
            p.x=tx; p.y=ty;
        }
        return;
    } else {
        return;
    }

    // Check if player walked into an enemy
    if(enemyAt(p.x,p.y)>=0) triggerDeath(playerID);
}

function checkCaveWon(){
    for(var id in pls){
        if(pls[id].exited){
            wonDelay=CAVE_WIN_DELAY;
            chat("Cave "+caveName+" cleared! Next cave in "+Math.ceil(CAVE_WIN_DELAY)+"s...");
            return;
        }
    }
}

// ============================================================
// Rendering
// ============================================================
function doRenderAscii(buf, playerID, ox, oy, width, height){
    var me=pls[playerID];
    var px=me?me.x:startX;
    var py=me?me.y:startY;
    var camX=clamp(px-Math.floor(width/2), 0, Math.max(0,CAVE_W-width));
    var camY=clamp(py-Math.floor(height/2), 0, Math.max(0,CAVE_H-height));

    // Build quick lookup maps
    var eMap={};
    for(var i=0;i<enemies.length;i++){
        var e=enemies[i];
        eMap[e.x+","+e.y]=e;
    }
    var pMap={};
    for(var id in pls){
        var p=pls[id];
        if(!p.exited) pMap[p.x+","+p.y]=id;
    }

    var blink=Math.floor(elapsed*2)%2===0;

    for(var sy=0;sy<height;sy++){
        for(var sx=0;sx<width;sx++){
            var gx=camX+sx, gy=camY+sy;
            if(!inBounds(gx,gy)){
                buf.setChar(sx,sy," ",null,null);
                continue;
            }
            var key=gx+","+gy;
            var ch=" ", fg=null, bg=null;

            // Base tile
            switch(grid[gy][gx]){
                case EMPTY:      ch=" "; break;
                case DIRT:       ch=":"; fg=C_DIRT_FG; bg=C_DIRT_BG; break;
                case WALL:       ch="#"; fg=C_WALL_FG; bg=C_WALL_BG; break;
                case BOULDER:    ch="O"; fg=C_BOULDER; break;
                case DIAMOND:    ch=blink?"*":"+"; fg=blink?C_DIA_A:C_DIA_B; break;
                case EXIT_C:     ch="+"; fg=C_EXIT_C; break;
                case EXIT_O:     ch=blink?"X":"O"; fg=C_EXIT_O; break;
                case AMOEBA:     ch="~"; fg=C_AMOEBA; break;
                case MAGIC_WALL: ch="M"; fg=C_MAGIC_FG; bg=C_MAGIC_BG; break;
            }

            // Overlay: enemy
            if(eMap[key]){
                var en=eMap[key];
                if(en.type==="firefly"){  ch="/"; fg=C_FIREFLY;   bg=null; }
                else {                    ch="%"; fg=C_BUTTERFLY; bg=null; }
            }

            // Overlay: player
            if(pMap[key]){
                var pid2=pMap[key];
                var p2=pls[pid2];
                if(p2.dead){ ch="x"; fg=C_DEAD_FG; bg=null; }
                else if(pid2===playerID){ ch="@"; fg=C_PLAYER; bg=null; }
                else { ch="@"; fg=C_OTHER; bg=null; }
            }

            buf.setChar(sx,sy,ch,fg,bg);
        }
    }
}

// ============================================================
// Results / player management
// ============================================================
function buildResults(){
    var arr=[];
    for(var id in pls){
        var p=pls[id];
        arr.push({name:p.name, score:p.score});
        if(p.score>highScore) highScore=p.score;
    }
    arr.sort(function(a,b){ return b.score-a.score; });
    var res=[];
    for(var i=0;i<arr.length;i++) res.push({name:arr[i].name, result:arr[i].score+" pts"});
    return res;
}

function addPlayer(id, name, team){
    if(pls[id]) return;
    pls[id]={
        name:name, team:team,
        x:startX, y:startY,
        lives:3, score:0, diamonds:0,
        exited:false, dead:false,
        respawnTimer:0, invulnTimer:INVULN_TIME
    };
    plOrder.push(id);
}

// ============================================================
// Game object
// ============================================================
var Game = {
    gameName:  "Boulder Dash",
    teamRange: { min:1, max:4 },

    load: function(savedState){
        if(savedState && savedState.highScore) highScore=savedState.highScore;
        var t=teams();
        var totalPlayers=0;
        for(var i=0;i<t.length;i++) totalPlayers+=t[i].players.length;
        Game.splashScreen =
            "=== BOULDER DASH ===\n" +
            "Dig through dirt, collect diamonds, reach the exit!\n\n" +
            "Controls: arrow keys to move / dig / push boulders\n" +
            "Collect enough diamonds to open the exit door [X]\n" +
            "Watch out for falling boulders and enemies!\n\n" +
            "  Firefly [/] — explodes into empty space when crushed\n" +
            "  Butterfly [%] — explodes into diamonds when crushed!\n" +
            "  Amoeba [~] — suffocate it for diamonds, or it turns to boulders\n" +
            "  Magic Wall [M] — boulders falling through become diamonds\n\n" +
            (highScore>0?"High score: "+highScore+"\n\n":"") +
            "Caves: "+CAVES.length+"   Players: "+totalPlayers;
    },

    begin: function(){
        caveIndex=0; elapsed=0; gameEnded=false;
        pls={}; plOrder=[];
        var t=teams();
        for(var i=0;i<t.length;i++){
            for(var j=0;j<t[i].players.length;j++){
                var p=t[i].players[j];
                addPlayer(p.id, p.name, t[i].name);
            }
        }
        loadCave(0);
        log("Boulder Dash started: cave 0, "+plOrder.length+" players");
    },

    onPlayerJoin: function(playerID){
        var t=teams();
        for(var i=0;i<t.length;i++){
            for(var j=0;j<t[i].players.length;j++){
                if(t[i].players[j].id===playerID){
                    addPlayer(playerID, t[i].players[j].name, t[i].name);
                    return;
                }
            }
        }
    },

    onPlayerLeave: function(playerID){
        delete pls[playerID];
        for(var i=0;i<plOrder.length;i++){
            if(plOrder[i]===playerID){ plOrder.splice(i,1); break; }
        }
    },

    onInput: function(playerID, key){
        var p=pls[playerID];
        if(!p||p.dead) return;
        if(key==="up")    tryMove(playerID, 0,-1);
        if(key==="down")  tryMove(playerID, 0, 1);
        if(key==="left")  tryMove(playerID,-1, 0);
        if(key==="right") tryMove(playerID, 1, 0);
    },

    update: function(dt){
        if(gameEnded) return;
        elapsed+=dt;
        timeLeft-=dt;

        // Cave-won delay: load next cave or end game
        if(wonDelay>0){
            wonDelay-=dt;
            if(wonDelay<=0){
                if(caveIndex+1>=CAVES.length){
                    gameEnded=true;
                    gameOver(buildResults(), {highScore:highScore});
                } else {
                    loadCave(caveIndex+1);
                }
            }
            return;
        }

        // Time expired
        if(timeLeft<=0){
            timeLeft=0; gameEnded=true;
            chat("Time's up!");
            gameOver(buildResults(), {highScore:highScore});
            return;
        }

        // All players out of lives
        var anyAlive=false;
        for(var id in pls){ if(pls[id].lives>0){ anyAlive=true; break; } }
        if(!anyAlive){
            gameEnded=true;
            gameOver(buildResults(), {highScore:highScore});
            return;
        }

        // Respawn / invulnerability timers
        for(var id in pls){
            var p=pls[id];
            if(p.dead && p.lives>0){
                p.respawnTimer-=dt;
                if(p.respawnTimer<=0){
                    p.dead=false;
                    p.x=startX; p.y=startY;
                    p.invulnTimer=INVULN_TIME;
                }
            }
            if(p.invulnTimer>0) p.invulnTimer-=dt;
        }

        // Magic wall countdown
        if(magicWallActive){
            magicWallTimer-=dt;
            if(magicWallTimer<=0){ magicWallActive=false; chat("Magic walls have expired!"); }
        }

        // Physics
        physicsTimer+=dt;
        while(physicsTimer>=PHYSICS_INTERVAL){
            physicsTimer-=PHYSICS_INTERVAL;
            physicsTick();
        }

        // Enemy movement
        enemyTimer+=dt;
        if(enemyTimer>=ENEMY_INTERVAL){
            enemyTimer-=ENEMY_INTERVAL;
            enemyTick();
        }

        // Amoeba
        if(amoebaList.length>0){
            amoebaTimer+=dt;
            if(amoebaTimer>=AMOEBA_INTERVAL){
                amoebaTimer-=AMOEBA_INTERVAL;
                amoebaGrow();
            }
        }
    },

    renderAscii: function(buf, playerID, ox, oy, width, height){
        doRenderAscii(buf, playerID, ox, oy, width, height);
    },

    statusBar: function(playerID){
        var p=pls[playerID];
        if(!p) return "Boulder Dash";
        var need=Math.max(0, diamondsNeeded-diamondsCollected);
        var t=Math.max(0, Math.ceil(timeLeft));
        var mw=magicWallActive?" [Magic:"+Math.ceil(magicWallTimer)+"s]":"";
        return "Cave "+caveName+": "+caveTitle+mw+
               "  | need:"+need+" got:"+p.diamonds+
               "  | lives:"+Math.max(0,p.lives)+
               "  | score:"+p.score+
               "  | "+t+"s";
    },

    commandBar: function(playerID){
        var p=pls[playerID];
        if(!p||p.dead) return "Boulder Dash — waiting to respawn...";
        return "[arrows] Move/Dig  [arrow into boulder] Push  [Enter] Chat";
    },

    suspend: function(){
        return {
            grid:grid, fallingGrid:fallingGrid,
            enemies:enemies, pls:pls, plOrder:plOrder,
            amoebaList:amoebaList, caveIndex:caveIndex,
            diamondsCollected:diamondsCollected, diamondsNeeded:diamondsNeeded,
            timeLeft:timeLeft, exitOpen:exitOpen, elapsed:elapsed,
            magicWallActive:magicWallActive, magicWallTimer:magicWallTimer
        };
    },

    resume: function(s){
        if(!s) return;
        caveData   = CAVES[s.caveIndex];
        caveIndex  = s.caveIndex;
        caveName   = caveData.name;
        caveTitle  = caveData.title;
        // Re-parse to get CAVE_W/H and startX/Y
        var parsed = parseGrid(caveData.raw);
        // CAVE_W and CAVE_H are set as side-effects of parseGrid
        startX=parsed.sx; startY=parsed.sy;
        // Restore dynamic state
        grid=s.grid; fallingGrid=s.fallingGrid;
        movedGrid=[];
        for(var y=0;y<CAVE_H;y++){
            movedGrid.push([]);
            for(var x=0;x<CAVE_W;x++) movedGrid[y].push(false);
        }
        enemies=s.enemies; pls=s.pls; plOrder=s.plOrder;
        amoebaList=s.amoebaList;
        diamondsCollected=s.diamondsCollected; diamondsNeeded=s.diamondsNeeded;
        timeLeft=s.timeLeft; exitOpen=s.exitOpen; elapsed=s.elapsed;
        magicWallActive=s.magicWallActive; magicWallTimer=s.magicWallTimer;
    },

    unload: function(){
        return { highScore:highScore };
    }
};
