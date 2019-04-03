create table if not exists notes (
  note_id blob primary key,
  note text not null
);