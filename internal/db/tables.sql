CREATE SEQUENCE public.users_id_seq
    START WITH 1
    NO MINVALUE
    NO MAXVALUE
;

CREATE TABLE public.users (
    id int8 PRIMARY KEY DEFAULT nextval('public.users_id_seq'),

    login    text NOT NULL,
    password text NOT NULL,

    create_ts int8 NOT NULL, -- unix microseconds timestamp

    UNIQUE (login)
);

CREATE TABLE public.sessions (
    token   text PRIMARY KEY,
    user_id int8 NOT NULL,

    create_ts int8 NOT NULL, -- unix microseconds timestamp
    expire_ts int8 NOT NULL, -- unix microseconds timestamp

    FOREIGN KEY (user_id) REFERENCES public.users(id)
);

CREATE SEQUENCE public.tasks_id_seq
    START WITH 1
    NO MINVALUE
    NO MAXVALUE
;

CREATE TABLE public.tasks (
    id      int8 PRIMARY KEY DEFAULT nextval('public.tasks_id_seq'),
    user_id int8 NOT NULL,

    state        int4 NOT NULL,
    importance   int4,
    progress     int4,
    max_progress int4,
    start_ts     int8,          -- unix microseconds timestamp    
    deadline_ts  int8,          -- unix microseconds timestamp
    title        text NOT NULL,
    description  text,

    create_ts int8 NOT NULL, -- unix microseconds timestamp
    update_ts int8,          -- unix microseconds timestamp

    FOREIGN KEY (user_id) REFERENCES public.users(id)
);

COMMENT ON COLUMN public.tasks.state IS
'Determines completeness and visibility to user.

0 - active
1 - done
2 - canceled
3 - expired
';

COMMENT ON COLUMN public.tasks.start_ts IS
'Planned start time. Tasks can be planned ahead of their actual
start (before meaningful can be done). Tasks which are planned
but not yet started are shown with lowest urgency.

NULL means that task is already started (even upon creation).
';


COMMENT ON COLUMN public.tasks.max_progress IS
'Maximum numerical progress for tasks which have progress indicator.

Equals NULL for tasks which do not have progress.
';

COMMENT ON COLUMN public.tasks.progress IS
'Current numerical progress for tasks which have progress indicator.

Equals NULL for tasks which do not have progress.
';

CREATE SEQUENCE public.tags_id_seq
    START WITH 1
    NO MINVALUE
    NO MAXVALUE
;

CREATE TABLE public.tags (
    id int8 PRIMARY KEY DEFAULT nextval('public.tags_id_seq'),

    name text NOT NULL,

    UNIQUE (name) 
);

CREATE TABLE public.task_tags (
    task_id int8 NOT NULL,
    tag_id  int8 NOT NULL,

    PRIMARY KEY (task_id, tag_id),

    FOREIGN KEY (task_id) REFERENCES public.tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES public.tags(id) ON DELETE CASCADE
);

CREATE SEQUENCE public.rep_tasks_id_seq
    START WITH 1
    NO MINVALUE
    NO MAXVALUE
;

CREATE TABLE public.rep_tasks (
    id int8 PRIMARY KEY DEFAULT nextval('public.rep_tasks_id_seq'),
    user_id int8 NOT NULL,

    state           int4  NOT NULL,
    type            int4  NOT NULL,
    time_limit      int8,           -- in microseconds
    next_trigger_ts int8,           -- unix microseconds timestamp
    lock_ts         int8,           -- unix microseconds timestamp
    timezone        text  NOT NULL,
    schedule        jsonb NOT NULL,

    importance   int4,
    max_progress int4,
    title        text NOT NULL,
    description  text,

    create_ts int8 NOT NULL, -- unix microseconds timestamp
    update_ts int8,          -- unix microseconds timestamp

    FOREIGN KEY (user_id) REFERENCES public.users(id)
);

COMMENT ON COLUMN public.rep_tasks.type IS
'Determines how task should be repeated.
Discriminator for json stored in schedule column.

0 - interval
1 - daily (can be configured to trigger once in set number of days)
2 - weekly (trigger at specific week day(s))
3 - monthly (trigger at specific month day(s))
4 - yearly
';

COMMENT ON COLUMN public.rep_tasks.state IS
'Determines suspension and visibility to user.

0 - active
1 - locked (assigned when task is being processed to avoid races)
2 - suspended
3 - canceled
';

CREATE SCHEMA notes;

CREATE SEQUENCE notes.topics_id_seq
    START WITH 1
    NO MINVALUE
    NO MAXVALUE
;

CREATE TABLE notes.topics (
    id      int8 PRIMARY KEY DEFAULT nextval('notes.topics_id_seq'),
    user_id int8 NOT NULL,
    "path"  text NOT NULL,

    title       text NOT NULL,
    description text,

    "offset"  int8 NOT NULL, -- offset of next message, starts with 0
    create_ts int8 NOT NULL, -- unix microseconds timestamp
    update_ts int8,          -- unix microseconds timestamp

    FOREIGN KEY (user_id) REFERENCES public.users(id),

    UNIQUE (user_id, "path")
);

CREATE SEQUENCE notes.messages_id_seq
    START WITH 1
    NO MINVALUE
    NO MAXVALUE
;

CREATE TABLE notes.messages (
    id       int8 PRIMARY KEY DEFAULT nextval('notes.messages_id_seq'),
    topic_id int8 NOT NULL,

    "text" text NOT NULL,

    "offset"  int8 NOT NULL, -- starts with 0 for each topic
    create_ts int8 NOT NULL, -- unix microseconds timestamp
    update_ts int8,          -- unix microseconds timestamp

    FOREIGN KEY (topic_id) REFERENCES notes.topics(id)
);
