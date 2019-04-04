drop table if exists notes;
drop table if exists authors;

create table authors (
  author_id   blob primary key,
  author_name text not null
);

create table notes (
  note_id   blob primary key,
  title     text not null,
  content   text not null,
  author_id text not null,
  foreign key(author_id) references authors(author_id)
);
