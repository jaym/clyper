BEGIN TRANSACTION;

CREATE TABLE `episodes` (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    season INT NOT NULL,
    episode INT NOT NULL
);

CREATE TABLE `thumbnails` (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    episode_id INT NOT NULL,
    storage_key VARCHAR(255) NOT NULL,
    start_ts INT NOT NULL,
    end_ts INT NOT NULL,
    FOREIGN KEY (episode_id) REFERENCES episodes(id)
);

CREATE TABLE `videos` (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    episode_id INT NOT NULL,
    storage_key VARCHAR(255) NOT NULL,
    FOREIGN KEY (episode_id) REFERENCES episodes(id)
);

CREATE TABLE `subtitles` (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    episode_id INT NOT NULL,
    start_ts INT NOT NULL,
    end_ts INT NOT NULL,
    text TEXT NOT NULL,
    FOREIGN KEY (episode_id) REFERENCES episodes(id)
);

CREATE VIRTUAL TABLE `subtitles_fts` USING fts5(
    text,
    content=`subtitles`,
);

CREATE TRIGGER `subtitles_ai` AFTER INSERT ON `subtitles`
BEGIN
    INSERT INTO `subtitles_fts` (rowid, text) VALUES (new.id, new.text);
END;

COMMIT;