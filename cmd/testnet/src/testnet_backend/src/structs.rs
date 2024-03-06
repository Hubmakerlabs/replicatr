use candid::{CandidType, Decode, Encode};
use ic_stable_structures::{storable::Bound, Storable};
use serde::{Deserialize, Serialize};
use std::{borrow::Cow, fmt::Debug};

#[derive(CandidType, Deserialize, Serialize, Debug, Clone)]
pub struct Event {
    pub id: String,
    pub pubkey: String,
    pub created_at: i64,
    pub kind: u16,
    pub tags: Vec<Vec<String>>,
    pub content: String,
    pub sig: String,
}

impl Storable for Event {
    fn to_bytes(&self) -> std::borrow::Cow<[u8]> {
        Cow::Owned(Encode!(self).unwrap())
    }

    fn from_bytes(bytes: std::borrow::Cow<[u8]>) -> Self {
        Decode!(bytes.as_ref(), Self).unwrap()
    }

    const BOUND: Bound = Bound::Unbounded;
}

impl Event {
    pub fn new(
        id: String,
        pubkey: String,
        created_at: i64,
        kind: u16,
        tags: Vec<Vec<String>>,
        content: String,
        sig: String,
    ) -> Self {
        Self {
            id,
            pubkey,
            created_at,
            kind,
            tags,
            content,
            sig,
        }
    }

    pub fn is_match(&self, filter: &Filter) -> bool {
        // ID filter
        if !filter.ids.is_empty() && !filter.ids.contains(&self.id) {
            return false;
        }

        // Kind filter
        if !filter.kinds.is_empty() && !filter.kinds.contains(&self.kind.into()) {
            return false;
        }

        // Author filter
        if !filter.authors.is_empty() && !filter.authors.contains(&self.pubkey) {
            return false;
        }

        // Tag filter
        if !filter.tags.is_empty()
            && filter.tags.iter().any(|tag_pair| {
                !self
                    .tags
                    .iter()
                    .any(|event_tag_vec| event_tag_vec.contains(&tag_pair.key))
            })
        {
            return false;
        }

        // Since filter
        if filter.since > 0 && self.created_at < filter.since {
            return false;
        }

        // Until filter
        if filter.until > 0 && self.created_at > filter.until {
            return false;
        }

        true
    }
}

// Assuming KeyValuePair looks like this, based on your corrected version:
#[derive(CandidType, Deserialize, Serialize, Debug, Clone)]
pub struct KeyValuePair {
    pub key: String,
    pub value: Vec<String>,
}

#[derive(CandidType, Deserialize, Serialize, Debug)]
pub struct Filter {
    pub ids: Vec<String>,
    pub kinds: Vec<u64>, // `nat16` in Candid
    pub authors: Vec<String>,
    pub tags: Vec<KeyValuePair>, // Corrected to match the new KeyValuePair definition
    pub since: i64,              // Candid `int`
    pub until: i64,              // Candid `int`
    pub limit: i64,              // Candid `int`
    pub search: String,
}
