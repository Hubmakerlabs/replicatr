use crate::{
    structs::{Event, Filter},
    EVENTS,
    MEMORY_MANAGER
};
use candid::export_service;
use ic_cdk_macros::{query, update};
use ic_stable_structures::StableBTreeMap;
use ic_stable_structures::memory_manager::MemoryId;
use crate::acl::{init,is_user};


#[ic_cdk_macros::init]
fn initialize() {
    init();
}


#[query(guard = "is_user")]
fn test() -> String {
    "Hello, world!".to_string()
}

#[query(guard = "is_user")]
fn _get_events() -> Vec<(String, Event)> {
    EVENTS.with(|events| {
        events
            .borrow()
            .iter()
            .map(|(k, v)| (k.clone(), v.clone()))
            .collect()
    })
}

#[query(guard = "is_user")]
fn _get_events_count() -> u64 {
    EVENTS.with(|events| events.borrow().len() as u64)
}

#[update(guard = "is_user")]
fn save_event(event: Event) -> String {
    let event_for_logging = event.clone();
    EVENTS.with(|events| {
        events.borrow_mut().insert(event.id.clone(), event);
    });
    // this only works on the local replica
    ic_cdk::println!("Saving record: {:?}", event_for_logging);
    "success".to_string()
}

#[update(guard = "is_user")]
fn save_events(events: Vec<Event>) -> String {
    let events_for_logging = events.clone();
    EVENTS.with(|events_map| {
        for event in events {
            events_map.borrow_mut().insert(event.id.clone(), event);
        }
    });
    // this only works on the local replica
    ic_cdk::println!("Saving records: {:?}", events_for_logging);
    "success".to_string()
}

#[update(guard = "is_user")]
fn delete_event(id: String) -> String {
    let event_for_logging = id.clone();
    EVENTS.with(|events| {
        events.borrow_mut().remove(&id);
    });

    // this only works on the local replica
    ic_cdk::println!("Deleting record: {:?}", event_for_logging);
    "success".to_string()
}

#[query(guard = "is_user")]
fn get_events(filter: Filter) -> Vec<Event> {
    let result = EVENTS.with(|events| {
        events
            .borrow()
            .iter()
            .filter(|(_, event)| event.is_match(&filter))
            .map(|(_, event)| event.clone())
            .collect()
    });

    // this only works on the local replica
    ic_cdk::println!("Query Results: {:#?}", result);
    result
}

#[query(guard = "is_user")]
fn get_events_count(filter: Filter) -> u64 {
    get_events(filter).len() as u64

    // ic_cdk::println!("Query Results: {:#?}", result);
}

#[query(name = "__get_candid_interface_tmp_hack")]
pub fn __export_did_tmp_() -> String {
    export_service!();
    __export_service()
}

#[update(guard = "is_user")]
fn clear_events() -> String {
    EVENTS.with(|events| {
        // Replace the contents of `events` with a new, empty `StableBTreeMap`.
        *events.borrow_mut() = StableBTreeMap::init(MEMORY_MANAGER.with(|p| p.borrow().get(MemoryId::new(0))));
    });

    ic_cdk::println!("All events have been cleared.");
    "All events have been cleared".to_string()
}



// Method used to save the candid interface to a file
#[test]
pub fn candid() {
    use std::env;
    use std::fs::write;
    use std::path::PathBuf;


    let dir = PathBuf::from(env::var("CARGO_MANIFEST_DIR").unwrap());
    let dir = dir.parent().unwrap().parent().unwrap().join("candid");
    write(
        dir.join(format!("testnet_backend.did")),
        __export_did_tmp_(),
    )
    .expect("Write failed.");
}
