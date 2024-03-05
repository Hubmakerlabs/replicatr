use candid::{CandidType, Deserialize as CandidDeserialize, Int, Nat};
use ic_cdk::api;
use ic_cdk_macros::{query, update};
use serde::{Serialize, Deserialize as SerdeDeserialize};
use std::cell::RefCell;
use std::fmt::Debug;
use std::collections::HashSet;


#[derive(CandidType,CandidDeserialize, Serialize, Debug, Clone)]
struct Event {
    id: String,
    pubkey: String,
    created_at: Int, // Using Int for compatibility with Candid's int type
    kind: u16,
    tags: Vec<Vec<String>>,
    content: String,
    sig: String,
}

// Assuming KeyValuePair looks like this, based on your corrected version:
#[derive(CandidType, CandidDeserialize, Serialize, Debug, Clone)]
struct KeyValuePair {
    key: String,
    value: Vec<String>,
}

#[derive(CandidType, CandidDeserialize, Serialize, Debug)]
struct Filter {
    ids: Vec<String>,
    kinds: Vec<u16>, // `nat16` in Candid
    authors: Vec<String>,
    tags: Vec<KeyValuePair>, // Corrected to match the new KeyValuePair definition
    since: Int, // Candid `int`
    until: Int, // Candid `int`
    limit: Int, // Candid `int`
    search: String,
}

thread_local! {
    // This will store the events. Adjust the type if your storage pattern differs.
    static EVENTS: RefCell<Vec<Event>> = RefCell::new(Vec::new());
}

#[update]
fn save_event(event: Event) -> String {
    EVENTS.with(|events| {
        events.borrow_mut().push(event);
        // Depending on your use case, you might want to log this operation,
        // validate the event, or send a confirmation.
    });
    "success".to_string()
}


#[query]
fn get_events(filter: Filter) -> Vec<Event> {
    EVENTS.with(|events| {
        events.borrow().iter().filter(|event| {

            // ID filter
            if !filter.ids.is_empty() && !filter.ids.contains(&event.id) {
                return false;
            }

            // Kind filter
            if !filter.kinds.is_empty() && !filter.kinds.contains(&event.kind) {
                return false;
            }

            // Author filter
            if !filter.authors.is_empty() && !filter.authors.contains(&event.pubkey) {
                return false;
            }

            // Tag filter (simplified for demonstration)
            if !filter.tags.is_empty() && filter.tags.iter().any(|tag_pair| {
                !event.tags.iter().any(|event_tag_vec| event_tag_vec.contains(&tag_pair.key))
            }) {
                return false;
            }

            // Since filter
            if filter.since >= 0 && event.created_at < filter.since {
                return false;
            }

            // Until filter
            if filter.until >= 0 && event.created_at > filter.until {
                return false;
            }
             true
        })
        .cloned()
        .collect()
    })
}


