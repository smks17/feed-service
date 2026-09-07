CREATE TABLE posts (
    id         BIGINT PRIMARY KEY,
    author_id  BIGINT      NOT NULL,
    content    TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX posts_author_created_idx ON posts (author_id, created_at DESC);
CREATE INDEX posts_created_idx ON posts (created_at DESC);

CREATE TABLE follows (
    follower_id  BIGINT NOT NULL,
    following_id BIGINT NOT NULL,
    PRIMARY KEY (follower_id, following_id)
);

CREATE TABLE likes (
    post_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    PRIMARY KEY (post_id, user_id)
);

CREATE TABLE processed_events (
    event_id     UUID PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
