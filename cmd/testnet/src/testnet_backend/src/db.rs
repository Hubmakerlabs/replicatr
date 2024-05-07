use crate::{
    structs::{Event, Filter},
    EVENTS,
    MEMORY_MANAGER
};
use candid::export_service;
use ic_cdk_macros::{query, update};
use ic_stable_structures::StableBTreeMap;
use ic_stable_structures::memory_manager::MemoryId;
use crate::acl::{is_user};








pub fn get_all_events_db() -> Vec<(String, Event)> {
    EVENTS.with(|events| {
        events
            .borrow()
            .iter()
            .map(|(k, v)| (k.clone(), v.clone()))
            .collect()
    })
}

pub fn count_all_events_db() -> u64 {
    EVENTS.with(|events| events.borrow().len() as u64)
}


pub fn save_event_db(event: Event) -> String {
    let event_for_logging = event.clone();
    EVENTS.with(|events| {
        events.borrow_mut().insert(event.id.clone(), event);
    });
    "success".to_string()
}


pub fn save_events_db(events: Vec<Event>) -> String {
    let events_for_logging = events.clone();
    EVENTS.with(|events_map| {
        for event in events {
            events_map.borrow_mut().insert(event.id.clone(), event);
        }
    });
    "success".to_string()
}


pub fn delete_event_db(id: String) -> String {
    let event_for_logging = id.clone();
    EVENTS.with(|events| {
        events.borrow_mut().remove(&id);
    });
    "success".to_string()
}


pub fn get_events_db(filter: Filter) -> Vec<Event> {
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


pub fn count_events_db(filter: Filter) -> u64 {
    get_events_db(filter).len() as u64
}


pub fn clear_events_db() -> String {
    EVENTS.with(|events| {
        // Replace the contents of `events` with a new, empty `StableBTreeMap`.
        *events.borrow_mut() = StableBTreeMap::init(MEMORY_MANAGER.with(|p| p.borrow().get(MemoryId::new(0))));
    });
    "All events have been cleared".to_string()
}
