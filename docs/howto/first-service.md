# First Microservice Walk-Through

This guide walks through the creation of a microservice that implements a simplified version of a word guessing game. The player needs to reveal a secret 5-letter word by entering limited number of guesses while observing how closely they match the secret word.

For example, if the secret word is `APPLE` and a guess is `OPERA`, the system will identify the second letter `P` as an exact match (green), the letters `A` and `E` as out-of-place matches (yellow), and the letters `O` and `R` as mismatches (grey).

<img src="first-service-1.png" width="213">

#### Step 1: Bootstrap

To get started, create a directory under `examples` to house the microservice files. We'll call our game `Wordly` so name the directory `examples/wordly`.

Next, create a new source file `doc.go` and place in it the `go:generate` directive and package name.

```go
//go:generate go run github.com/microbus-io/fabric/codegen

package wordly
```

From within the directory, run `go generate`.

```cmd
cd examples/wordly
go generate
```

A `service.yaml` template is generated. We'll fill it in next.

#### Step 2: Service Definition

Open `service.yaml`. This is your starting point and where you declare the structure of the microservice. First, name the service in the `general:` section:

```yaml
general:
  host: wordly.example
  description: Wordly is a word guessing game.
```

`Wordly` is a simple game and it requires a single endpoint `/play`. We'll use a `web` handler because we want to create an HTML interface.

```yaml
webs:
  - signature: Play()
    description: Play a game of Wordly.
```

Finally, we'll want to control the maximum number of guesses using a config.

```yaml
configs:
  - signature: MaxGuesses() (max int)
    description: MaxGuesses to guess the secret word.
    default: 6
    validation: int [1,]
```

Run `go generate` again to generate the boilerplate code. You'll see quite a few [source files created](../blocks/uniform-code.md) in the `examples/wordly` directory. We will mostly touch `service.go`.

```
wordly
├── app
│   └── wordly
│       └── main-gen.go
├── intermediate
│   ├── intermediate-gen.go
│   └── mock-gen.go
├── resources
│   └── embed-gen.go
├── wordlyapi
│   └── client-gen.go
├── doc.go
├── integration_test.go
├── integration-gen_test.go
├── service-gen.go
├── service.go
├── service.yaml
├── version-gen_test.go
└── version-gen.go
```

#### Step 3: Keeping Track of Games

Open `service.go`. You'll see a type definition for the `Service` struct and 3 empty functions: the standard lifecycle callbacks `OnStartup` and `OnShutdown`, and the endpoint `Play` that you defined in `service.yaml`.

We'd like our microservice to serve many users at the same time so we're going to have to track multiple games in parallel. The structure `Game` will maintain the status of a game.

```go
type Game struct {
    secretWord string
    guesses []string
}
```

A `map[string]*Game` will keep track of all games, indexed by the game ID.

```go
type Service struct {
	*intermediate.Intermediate // DO NOT REMOVE

	games map[string]*Game
}
```

This map needs to be initialized in `OnStartup`.

```go
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	svc.games = map[string]*Game{}
	return nil
}
```

#### Step 4: Implementation

We'll implement the game logic in the `Play` method which handles the `GET /play` request. This endpoint accepts two optional query arguments. The first, `game`, identifies the game. If it is not present, a new game is created. Add the following code to `Play`.

```go
gameID := r.URL.Query().Get("game")
game := svc.games[gameID]
if gameID == "" || game == nil {
    gameID = rand.AlphaNum64(12)
    game = &Game{
        secretWord: "APPLE",
    }
    svc.games[gameID] = game
}
```

The secret word is hard-coded to `APPLE` for the time being. We'll get to it later.

The second query argument is `guess`, which if present submits a guess to the identified game. Add the following code to `Play`.

```go
guess := r.URL.Query().Get("guess")
guess = strings.ToUpper(guess)
re := regexp.MustCompile("[A-Z]{5}")
if guess != "" && !re.MatchString(guess) {
    return errors.New("invalid guess")
}
if guess != "" && len(game.guesses) < svc.MaxGuesses() {
    game.guesses = append(game.guesses, guess)
}
```

Notice the use of `svc.MaxGuesses()` to pull the value of the config property that you defined earlier in `service.yaml`.

Next, we'll render the UI. First, a simple form to submit a guess,

```go
var page bytes.Buffer
page.WriteString(`<html><body>`)

page.WriteString(`
    <form method="GET">
        <input type="text" name="guess" maxlength="5">
        <input type="hidden" name="game" value="` + gameID + `">
        <input type="submit" value="Guess">
    </form><p></p>
`)
```

then all prior guesses with indication of how they match the secret word.

```go
for _, g := range game.guesses {
    secret := []rune(game.secretWord)
    word := []rune(g)
    colors := make([]string, 5)
    for i := 0; i < 5; i++ {
        colors[i] = "grey"
    }
    for i := 0; i < 5; i++ {
        if word[i] == secret[i] {
            colors[i] = "green"
            word[i] = 0
            secret[i] = 0
        }
    }
    for i := 0; i < 5; i++ {
        if word[i] == 0 {
            continue
        }
        for j := 0; j < 5; j++ {
            if secret[j] == 0 {
                continue
            }
            if word[i] == secret[j] {
                colors[i] = "#DBA800"
                word[i] = 0
                secret[j] = 0
                break
            }
        }
    }
    word = []rune(g)
    for i := 0; i < 5; i++ {
        page.WriteString(`<div style="background:` + colors[i] + `;color:white;`)
        page.WriteString(`display:inline-block;font-size:24pt;width:32pt;text-align:center;">`)
        page.WriteRune(word[i])
        page.WriteString(`</div>`)
    }
    page.WriteString(`<p>`)
}

page.WriteString(`</body></html>`)

w.Header().Set("Content-Type", "text/html")
w.Write(page.Bytes())
return nil  
```

#### Step 5: Randomized Secret Word

In the prior step we hard-coded the word `APPLE` as the secret word. We need to randomize it but we can't just generate it from random letters because that will result in unguessable gibberish. Instead, we'll choose a word randomly from a list of real dictionary words.

The `resources` folder allows us to embed resource files with the microservice. Create a new file `words.txt` in `examples/wordly/resources` and fill it with the following list of words. Remove any trailing spaces or newlines at the end of the file.

```
aback abase abate abaya abbey abbot abets abhor abide abode abort about above abuse abuts abyss ached aches acids acing ackee acorn acres acrid acted actin actor acute adage adapt added adder addle adept adieu adios adits adman admin admit adobe adobo adopt adore adorn adult adzes aegis aeons aerie affix afire afoot afore after again agape agate agave agent aggro agile aging aglow agony agora agree ahead ahold aided aider aides ailed aimed aimer aioli aired aisle alarm album alder aleph alert algae algal alias alibi alien align alike alive alkyd alkyl allay alley allot allow alloy allyl aloes aloft aloha alone along aloof aloud alpha altar alter altos alums amass amaze amber ambit amble ambos amend amide amine amino amiss amity amnio among amour amped ample amply amuse ancho angel anger angle angry angst anima anime anion anise ankle annas annex annoy annul anode anole antic antis antsy anvil aorta apace apart aphid apnea apple apply apron apses apter aptly aquas arbor ardor areal areas areca arena argon argot argue argus arias arils arise armed armor aroma arose array arrow arses arson artsy asana ascot ashen ashes aside asked asker askew aspen aspic assay asses asset aster astir asura atlas atman atoll atoms atone atopy attic audio audit auger aught augur aunts aunty aural auras autos auxin avail avers avert avian avoid avows await awake award aware awash awful awoke axels axial axils axing axiom axion axles axons azide azole azure babel babes babka backs bacon baddy badge badly bagel baggy bails bairn baits baize baked baker bakes baldy baled baler bales balks balky balls balms balmy balsa banal bands bandy banes bangs banjo banks barbs bards bared barer bares barge barks barmy barns baron barre basal based baser bases basic basil basin basis basks basso bassy baste batch bated bathe baths batik baton batts batty bawdy bawls bayed bayou beach beads beady beaks beams beamy beans beard bears beast beats beaus beaut beaux bebop becks beech beefs beefy beeps beers beery beets befit began beget begin begun beige being belay belch belie belle bells belly below belts bench bends bendy bento bents beret bergs berms berry berth beryl beset bests betas betel betta bevel bezel bhaji bible bicep biddy bided bides bidet bight bigot bijou biked biker bikes biles bilge bills billy bimbo bindi binds binge bingo biome biota bipod birch birds birth bison bitch biter bites bitsy bitty black blade blame bland blank blare blase blast blaze bleak bleat blebs bleed bleep blend bless blimp blind bling blini blink blips bliss blitz bloat blobs block blocs blogs bloke blond blood bloom bloop blots blown blows blued blues bluey bluff blunt blurb blurs blurt blush board boars boast boats bobby bocce boche boded bodes boffo bogey boggy bogie bogus boils bolas boles bolls bolts bolus bombe bombs bonds boned boner bones boney bongo bongs bonks bonny bonus boobs booby booed books booms boomy boons boors boost booth boots booty booze boozy boppy borax bored borer bores boric borne boron bosom boson bossy bosun botch bough boule bound bouts bowed bowel bower bowls boxed boxer boxes boyar boyos bozos brace bract brads brags braid brain brake brand brans brash brass brats brave bravo brawl brawn brays braze bread break bream breed brews briar bribe brick bride brief brier brigs brims brine bring brink briny brisk brits broad broch broil broke brome bronc brood brook broom broth brown brows bruin bruit brunt brush brute bubba bucks buddy budge buffs buggy bugle build built bulbs bulge bulks bulky bulla bulls bully bumps bumpy bunch bunds bundt bunks bunny bunts buoys burbs burgs burka burly burns burnt burps burqa burro burrs bursa burst bused buses bushy busts busty butch butte butts buxom buyer buzzy bylaw byres bytes byway cabal cabby caber cabin cable cacao cache cacti caddy cadet cadre cafes caged cages cagey cairn caked cakes cakey calfs calif calla calls calms calve calyx camel cameo campo camps campy canal candy caned canes canid canna canny canoe canon canto caped caper capes capon capos caput carat carbo carbs cards cared carer cares cargo carob carol carom carps carry carte carts carve cased cases casks caste casts catch cater catty caulk cause caved caver caves cavil cease cecal cecum cedar ceded cedes ceili celeb cello cells celts cents chads chafe chaff chain chair chalk champ chana chant chaos chaps chard charm chars chart chase chasm chats cheap cheat check cheek cheep cheer chefs chemo chert chess chest chews chewy chica chick chico chide chief child chile chili chill chime chimp china chine ching chino chins chips chirp chits chive chock choir choke chomp chops chord chore chose chows chubs chuck chuff chugs chump chums chunk churn chute cider cigar cinch circa cisco cited cites civet civic civil civvy clack clade claim clamp clams clang clank clans claps clash clasp class clave claws clays clean clear cleat clefs cleft clerk click cliff climb clime cline cling clink clips cloak clock clods clogs clomp clone close cloth clots cloud clout clove clown clubs cluck clued clues clump clung clunk coach coals coast coati coats cobia cobra cocci cocks cocky cocoa codas codec coded coder codes codex codon coeds cohos coifs coils coins cokes colas colds coles colic colin colon color colts comas combo combs comer comes comet comfy comic comma commo compo comps comte conch condo coned cones conga congo conic conks cooed cooks cools coops coopt coped copes copra copse coral cords cored corer cores corgi corks corky corms corns cornu corny corps costs cotta couch cough could count coupe coups court coven cover coves covet covey cowed cower cowls coyly crabs crack craft crags cramp crams crane crank crape craps crash crass crate crave crawl craws craze crazy creak cream credo creed creek creel creep creme crepe crept cress crest crews cribs crick cried crier cries crime crimp crisp crits croak crock crocs croft crone crony crook croon crops cross croup crowd crown crows crude cruel cruet crumb cruse crush crust crypt cubby cubed cubes cubic cubit cuddy cuffs culls culpa cults cumin cupid cuppa curbs curds cured cures curia curio curls curly curry curse curve curvy cushy cusps cuter cutie cutis cutup cycad cycle cyclo cynic cysts czars dacha daddy dados daffy daily dairy daisy dales dames damns damps dance dandy dared dares darks darns darts dashi dated dater dates datum daubs daunt davit dawns dazed deals dealt deans dears deary death debit debts debug debut decaf decal decay decks decor decoy decry deeds deems deeps deers defer deify deign deism deist deity dekes delay delft delis dells delta delve demon demos demur denim dense dents depot depth derby desks deter detox deuce devil dewar dhikr dhows dials diary diced dices dicey dicky dicta diets digit diked dikes dills dilly dimer dimes dimly dinar dined diner dines dingo dings dingy dinks dinky dinos diode dippy direr dirge dirty disco discs dishy disks ditch ditsy ditto ditty ditzy divan divas dived diver dives divot divvy dizzy docks dodge dodgy dodos doers doffs doges doggy dogma doing doled doles dolls dolly dolor dolts domed domes donee dongs donna donor donut dooms doomy doors doozy doped dopes dopey dorks dorky dorms dosas dosed doses doted dotes dotty doubt dough doula douse doves dowdy dowel dower downs downy dowry dowse doyen dozed dozen dozer dozes drabs draft drags drain drake drama drams drank drape drawl drawn draws drays dread dream dreck dregs dress dribs dried drier dries drift drill drily drink drips drive droid droll drone drool droop drops dross drove drown drugs druid drums drunk drupe dryad dryer dryly duals ducal ducat duchy ducks ducky ducts dudes duels duets duffs dukes dulls dully dulse dumbo dummy dumps dumpy dunce dunes dunks duomo duped dupes dural durum dusks dusky dusts dusty dutch duvet dwarf dweeb dwell dwelt dyads dyers dying dykes eager eagle eared earls early earns earth eased easel easer eases eaten eater eaves ebbed ebony ebook echos eclat edema edged edger edges edict edify edits eejit eerie egged egret eider eidos eight eject ejido eland elbow elder elect elegy elide elite elope elude elute elven elves email embed ember emcee emery emirs emits emote empty enact ended endow enema enemy enjoy ennui enoki enrol ensue enter entry envoy eosin epics epoch epoxy equal equip erase erect ergot erode erred error erupt essay ether ethic ethos ethyl etude euros evade evens event every evict evils evoke ewers exact exalt exams excel execs exert exile exist exits expat expel expos extol extra exude exult exurb eying eyrie fable faced facer faces facet facia facts faded fader fades faery fails faint fairs fairy faith faked faker fakes fakie fakir falls famed fancy fangs fanny farce fared fares farms farts fasts fatal fated fates fatso fatty fatwa fault fauna fauns favas faves favor fawns faxed faxes fazed fazes fears feast feats fecal feces feeds feels feign feint fella fells felon felts femme femur fence fends feral feria ferns ferny ferry fests fetal fetch feted fetes fetid fetus feuds fever fewer fiats fiber fibre fiche ficus fiefs field fiend fiery fifes fifth fifty fight filch filed filer files filet fills filly films filmy filth final finca finch finds fined finer fines finis finks fiord fired fires firms first fishy fists fitly fiver fives fixed fixer fixes fizzy fjord flack flags flail flair flake flaky flame flank flans flaps flare flash flask flats flaws flays fleas fleck flees fleet flesh flick flier flies fling float flood floor flour flown flows fluid flyer focal focus folks fonts foods force forms forth forty forum found frame fraud fresh fried fries front frost fruit fuels fully funds funny gains games gamma gases gates gauge gears genes genre ghost giant gifts girls given gives gland glass globe glory gloss glove glued goals goats going goods grace grade grain grams grand grant grape graph grasp grass grave great greek green greet grief grill grind grips gross group grove grown grows guard guess guest guide guild guilt habit hairs halls hands handy hangs happy harsh hated hates haven hawks heads heard heart heavy hedge heels hello helps hence herbs highs hills hints hired hobby holds holes holly homes honey honor hooks hoped hopes horns horse hosts hotel hours house hover human humor hurts icons ideal ideas idiot image imply inbox incur index indie inner input intro issue items jeans jelly jewel joins joint jokes judge juice juicy jumps keeps kicks kills kinda kinds kings knees knife knock knots known knows label labor lacks lakes lamps lands lanes large laser lasts later laugh layer leads leaks learn lease least leave legal lemon level lever light liked likes limbs limit lined linen liner lines links lions lists lived liver lives loads loans lobby local locks lodge logic logos looks loops loose lords loses loved lover loves lower loyal lucky lunar lunch lungs lying macro magic major maker makes males maple march marks marry masks match mates maths matte maybe mayor meals means meant meats medal media meets melee menus mercy merge merit merry messy metal meter metro micro midst might miles minds mines minor minus mixed mixer mixes model modem modes moist money month moral motor mount mouse mouth moved moves movie music myths nails naked named names nasal nasty naval needs nerve never newer newly nexus nicer niche night ninja ninth noble nodes noise noisy norms north notch noted notes novel nurse nylon oasis occur ocean offer often older olive omega onion onset opens opera opted optic orbit order organ other ought ounce outer owned owner oxide packs pages pains paint pairs panel panic pants paper parks parts party pasta paste patch paths patio pause peace peach peaks pearl pedal peers penis penny perks pests petty phase phone photo piano picks piece piles pills pilot pinch pipes pitch pixel pizza place plain plane plans plant plate plays plaza plots plugs poems point poker polar poles polls pools porch pores ports posed poses posts pouch pound power press price pride prime print prior prize probe promo prone proof props proud prove proxy psalm pulls pulse pumps punch pupil puppy purse queen query quest queue quick quiet quilt quite quote races racks radar radio rails rainy raise rally ranch range ranks rapid rated rates ratio razor reach react reads ready realm rebel refer reign relax relay renal renew reply reset resin retro rider rides ridge rifle right rigid rings rinse risen rises risks risky rival river roads robot rocks rocky rogue roles rolls roman rooms roots ropes roses rough round route royal rugby ruins ruled ruler rules rural sadly safer salad sales salon sandy satin sauce saved saves scale scalp scans scare scarf scary scene scent scoop scope score scout scrap screw seals seams seats seeds seeks seems sells sends sense serum serve setup seven sewer shade shaft shake shall shame shape share shark sharp sheep sheer sheet shelf shell shift shine shiny ships shirt shock shoes shook shoot shops shore short shots shown shows sides siege sight sigma signs silly since sites sixth sized sizes skies skill skins skirt skull slate slave sleek sleep slept slice slide slope slots small smart smell smile smoke snack snake sneak socks soils solar solid solve songs sonic sorry sorts souls sound south space spare spark speak specs speed spell spend spent sperm spice spicy spike spine spite split spoke spoon sport spots spray spurs squad stack staff stage stain stake stamp stand stark stars start state stats stays steak steal steam steel steep steer stems steps stick stiff still stock stole stone stood stool stops store storm story stove strap straw strip stuck study stuff style sucks sugar suite suits sunny super surge sushi swear sweat sweet swept swift swing swiss sword syrup table taken takes tales talks tanks tapes tasks taste tasty taxes teach teams tears teens teeth tells tempo tends tenth tents terms tests texts thank theft their theme there these thick thief thigh thing think third those three threw throw thumb tiger tight tiles timer times tired tires title toast today token tones tools tooth topic torch total touch tough tours towel tower towns toxic trace track tract trade trail train trait trans traps trash treat trees trend trial tribe trick tried tries trips trout truck truly trump trunk trust truth tubes tumor tuned tunes turbo turns tutor tweet twice twins twist types tyres ultra uncle under union unite units unity until upper upset urban urged urine usage users using usual vague valid value valve vapor vault vegan veins vents venue verse video views villa vinyl viral virus visas visit vital vivid vocal vodka voice volts voted voter votes wages wagon waist walks walls wants warns waste watch water watts waves wears weeds weeks weigh weird wells welsh whale wheat wheel where which while white whole whose wider widow width winds wines wings wiped wired wires witch wives woman women woods words works world worms worry worse worst worth would wound wrath wrist write wrong wrote yacht yards years yeast yield young yours youth yummy zones
```

With `svc.ReadResFile()`, it's a simple matter of choosing a random word from the resource.

```go
func (svc *Service) randomWord() string {
	binary, _ := svc.ReadResFile("words.txt")
	words := strings.Split(string(binary), " ")
	return strings.ToUpper(words[rand.Intn(len(words))])
}
```

Lastly, adjust the new game generation to make use of `randomWord` instead of the hard-coded `"APPLE"`.

```go
game = &Game{
    secretWord: svc.randomWord(),
}
```

#### Step 6: Time to Play!

Include the new microservice in the app in `main/main.go` and run it using `go run` or your IDE.

```go
func main() {
	app := application.New()
	app.Add(
		configurator.NewService(),
	)
	app.Add(
        // ...
    	wordly.NewService(), // <-- Add
	)
	app.Add(
		httpingress.NewService(),
	)
	app.Run()
}
```

Go to http://localhost:8080/wordly.example/play and enjoy!

Want to make the game harder by decreasing the number of guesses? Change the value of the config property `MaxGuesses` by editing `main/config.yaml`.

```yaml
wordly.example:
  MaxGuesses: 5
```

#### Step 7: Extra Credit

This microservice is far from being polished. Try the following on your own:

* Add a title to the top of the page
* Add instructions that mention how many guesses the player has left
* Add a UI element (link or button) to enable the player to start a new game
* Give a cleaner error message to the user on an invalid guess
* Do not print the guessing form if the player exhausted all of their guesses
* Do not accept guesses after the player identified the secret word
* Accept guesses only if they are themselves valid words
* Print the secret word when the player fails to guess it
* Support 4, 6 and 7 letter words
* The microservice stores state in local memory which will break when a second replica is added to the app and requests are load-balanced. Use the [distributed cache](./structure/dlru.md) to work around that
