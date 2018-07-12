
CREATE TABLE classes (
  id integer PRIMARY KEY NOT NULL,
  mask integer NOT NULL,
  powertype text NOT NULL,
  name text NOT NULL
);

CREATE TABLE class_colors (
  id SERIAL PRIMARY KEY,
  class_id integer NOT NULL,
  color text NOT NULL,
  FOREIGN KEY (class_id) REFERENCES classes(id)
);

CREATE TABLE races (
  id integer PRIMARY KEY NOT NULL,
  mask integer NOT NULL,
  side text NOT NULL,
  name text NOT NULL
);

CREATE TABLE toon (
  id SERIAL PRIMARY KEY NOT NULL,
  name text NOT NULL,
  race_id integer NOT NULL,
  class_id integer NOT NULL,
  gender integer NOT NULL,
  realm text NOT NULL,
  region text NOT NULL,
  UNIQUE (name, realm),
  FOREIGN KEY (class_id) REFERENCES classes(id),
  FOREIGN KEY (race_id) REFERENCES races(id)
);

CREATE TABLE stats (
  id SERIAL PRIMARY KEY NOT NULL ,
  toon_id integer NOT NULL,
  last_modified bigint,
  create_date date DEFAULT now(),
  achievement_points integer,
  number_exalted integer,
  mounts_owned integer,
  quests_completed integer,
  fish_caught integer,
  pets_owned integer,
  pet_battles_won integer,
  pet_battles_pvp_won integer,
  level integer,
  item_level integer,
  honorable_kills integer,
  UNIQUE (last_modified, toon_id),
  FOREIGN KEY (toon_id) REFERENCES toon(id)
);


INSERT INTO classes VALUES (1, 1, 'rage', 'Warrior');
INSERT INTO classes VALUES (2, 2, 'mana', 'Paladin');
INSERT INTO classes VALUES (3, 4, 'focus', 'Hunter');
INSERT INTO classes VALUES (4, 8, 'energy', 'Rogue');
INSERT INTO classes VALUES (5, 16, 'mana', 'Priest');
INSERT INTO classes VALUES (6, 32, 'runic-power', 'Death Knight');
INSERT INTO classes VALUES (7, 64, 'mana', 'Shaman');
INSERT INTO classes VALUES (8, 128, 'mana', 'Mage');
INSERT INTO classes VALUES (9, 256, 'mana', 'Warlock');
INSERT INTO classes VALUES (10, 512, 'energy', 'Monk');
INSERT INTO classes VALUES (11, 1024, 'mana', 'Druid');
INSERT INTO classes VALUES (12, 2048, 'fury', 'Demon Hunter');

INSERT INTO races VALUES (1, 1, 'alliance', 'Human');
INSERT INTO races VALUES (2, 2, 'horde', 'Orc');
INSERT INTO races VALUES (3, 4, 'alliance', 'Dwarf');
INSERT INTO races VALUES (4, 8, 'alliance', 'Night Elf');
INSERT INTO races VALUES (5, 16, 'horde', 'Undead');
INSERT INTO races VALUES (6, 32, 'horde', 'Tauren');
INSERT INTO races VALUES (7, 64, 'alliance', 'Gnome');
INSERT INTO races VALUES (8, 128, 'horde', 'Troll');
INSERT INTO races VALUES (9, 256, 'horde', 'Goblin');
INSERT INTO races VALUES (10, 512, 'horde', 'Blood Elf');
INSERT INTO races VALUES (11, 1024, 'alliance', 'Draenei');
INSERT INTO races VALUES (22, 2097152, 'alliance', 'Worgen');
INSERT INTO races VALUES (24, 8388608, 'neutral', 'Pandaren');
INSERT INTO races VALUES (25, 16777216, 'alliance', 'Pandaren');
INSERT INTO races VALUES (26, 33554432, 'horde', 'Pandaren');
INSERT INTO races VALUES (27, 67108864, 'horde', 'Nightborne');
INSERT INTO races VALUES (28, 134217728, 'horde', 'Highmountain Tauren');
INSERT INTO races VALUES (29, 268435456, 'alliance', 'Void Elf');
INSERT INTO races VALUES (30, 536870912, 'alliance', 'Lightforged Draenei');

INSERT INTO class_colors VALUES (1, 1, '#C79C6E');
INSERT INTO class_colors VALUES (2, 2, '#F58CBA');
INSERT INTO class_colors VALUES (3, 3, '#ABD473');
INSERT INTO class_colors VALUES (4, 4, '#FFF569');
INSERT INTO class_colors VALUES (5, 5, '#F0EBE0');
INSERT INTO class_colors VALUES (6, 6, '#C41F3B');
INSERT INTO class_colors VALUES (7, 7, '#0070DE');
INSERT INTO class_colors VALUES (8, 8, '#69CCF0');
INSERT INTO class_colors VALUES (9, 9, '#9482C9');
INSERT INTO class_colors VALUES (10, 10, '#00FF96');
INSERT INTO class_colors VALUES (11, 11, '#FF7D0A');
INSERT INTO class_colors VALUES (12, 12, '#A330C9');
