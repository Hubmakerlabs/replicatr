use candid::{CandidType, Deserialize as CandidDeserialize, Int};
use ic_cdk_macros::{query, update};
use serde::Serialize;
use std::cell::RefCell;
use std::fmt::Debug;
use num_traits::ToPrimitive;
use ic_cdk::println;



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
    static EVENTS: RefCell<Vec<Event>> = RefCell::new(Vec::new());
}

#[update]
fn save_event(event: Event) -> String {
    let event_for_logging = event.clone();
    EVENTS.with(|events| {
        events.borrow_mut().push(event);
    
    });
    ic_cdk::println!("Saving record: {:?}", event_for_logging);
    "success".to_string()
}

fn convert_int_to_usize(int_val: &Int) -> usize{
    let big_int: &num_bigint::BigInt = &int_val.0;
    match big_int.to_usize(){
        Some(val) => val,
        None => 500usize,
    }
}


#[query]
fn get_events(filter: Filter) -> Vec<Event> {
    let result = EVENTS.with(|events| {
        let limit = convert_int_to_usize(&filter.limit);
        let zero = Int::from(0);

        
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

            // Tag filter
            if !filter.tags.is_empty() && filter.tags.iter().any(|tag_pair| {
                !event.tags.iter().any(|event_tag_vec| event_tag_vec.contains(&tag_pair.key))
            }) {
                return false;
            }

            // Since filter
            if filter.since >= zero && event.created_at < filter.since {
                return false; 
            }

            // Until filter
            if filter.until >= zero && event.created_at > filter.until {
                return false;
            }

            true
        })
        .take(limit)
        .cloned()
        .collect()
    });

    ic_cdk::println!("Query Results: {:#?}",result);
    result
}


