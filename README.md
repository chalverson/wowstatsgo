Go program to historically record World of Warcraft character stats. Blizzard
provides an API but it does not have any historical data, the API just
returns what is there "now". I kind of like to see trends over time, so I need
to record things over time. It just records stats that I'm interested in, there are
many more that could be added, you would just need to change the API call and the database.

This is a rewrite of a Python app that was a rewrite of a Groovy app that all did
the same thing (more or less). I did this as a first "real" application in Go 
since it was a known quantity I could verify things were working and I knew the 
problem domain ahead of time.

This may not be totally idiomatic Golang, but hopefully it's pretty close. It's
longer than the Python version, but I'm actually doing error checking here and 
using structs, where the Python version is just Python, so not doing anything
type-wise or anything. I also got down some rabbit holes and added functionality,
so length of code is not a good comparison.

### Requirements

#### Go Packages

    go get -u github.com/jinzhu/gorm
    go get -u github.com/adrg/xdg
    go get -u github.com/spf13/viper
    go get -u github.com/tidwall/gjson
    go get -u gopkg.in/go-resty/resty.v1
    go get -u github.com/jessevdk/go-flags

#### Database

Requires either a Postgres or MariaDB database.

For Postgres:

    % createuser -W wowstats
    % createdb -O wowstats wowstats
    
For MariaDB (MySQL), get into MySQL as admin:

    mysql> create database wowstats;
    mysql> grant all privielges on wowstats.* to 'DBUSER'@'%' identified by 'DBPASSWORD'
    mysql> \q
   
The SQL will prepopulate some values from Blizzard so you do some queries and get
class and race information. These are hardcoded to use the `id` from Blizzard so that 
you can use the information that the API returns to figure things out. It also populates
the class colors (in HTML type format) as defined at:

https://wow.gamepedia.com/Class_colors
 
https://chasechristian.com/blog/2015/08/the-history-of-wow-class-colors/

They're not really used anywhere, but are available if desired. The Priest color is turned slightly
non-white so that it actually will show up.

### Configuration

#### Blizzard API Key

You will need a Blizzard API key from: https://dev.battle.net/ 

#### Configuration file

You will need to create a config file. By default it looks in XDG standard `~/.config/wowstats/wowstats.yml`. 

It should look similar to:

    dbDriver: postgres
    dbUrl: postgres://DBUSERNAME:DBPASSWORD@DBHOST/DBNAME?sslmode=disable
    apiKey: YOUR_API_KEY
    email:
      toAddress:
        - your@email.address
        - email2@test.com
      fromAddress: your@email.address
      server: localhost:25
    archiveDir: /home/username/.config/wowstats/json

For MySQL, you'll probably want something like:

    dbDriver: mysql
    dbUrl: DBUSERNAME:DBPASSWORD@tcp(DBHOST)/DBNAME?parseTime=true
    

* dbDriver - Database driver. Currently allowed values are `postgres` and `mysql`

* dbUrl - Database URL to use to connect to the database

* apiKey - Your Blizzard API key

* archiveStats - `true` or `false` depending on if you want to save the json

* archiveDir - Directory to store archived JSON files. This is optional and defaults to `$HOME/.local/share/wowstats/json`

* email - Top level email settings

    * toAddress - Can be multiple email addresses

    * fromAddress - Address email should come from

    * server - Email server to connect to in order to send email
    
### Usage

You will first need to add some characters, you do this by running with the `--add` flag:

    wowstats --add
    
The program will prompt you for some values and will then ask for confirmation and then add it to the
database. Repeat as needed to add characters.

If run without arguments, it will update the stats for every character in the database. It will log some
output which can be suppressed with the `--quiet` flag.

To get a quick summary use the `--summary` flag. This will output character level and item level for each
character in the database in a tabular format to STDOUT.

If run with `--emailsummary` it will do the same stats as `--summary` but will format it as an HTML
table and email it to the addresses listed in the configuration file.

You'd most likely run this via cron to do updates and then maybe on the next minute do a `--emailsummary`
call to see the status in your email. I just do it once a day because things don't change all that often.
