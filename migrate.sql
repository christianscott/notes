drop table if exists notes;

create table notes (
  note_id blob primary key,
  title   text not null,
  content text not null
);
