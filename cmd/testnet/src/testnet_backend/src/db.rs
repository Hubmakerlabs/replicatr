use crate::{
    structs::{Event, Filter},
    EVENTS,
    MEMORY_MANAGER
};
use ic_stable_structures::StableBTreeMap;
use ic_stable_structures::memory_manager::MemoryId;









pub fn get_all_events_db() -> Result<Vec<(String, Event)>,String> {
    let e : Vec<(String, Event)>= EVENTS.with(|events| {
        events
            .borrow()
            .iter()
            .map(|(k, v)| (k.clone(), v.clone()))
            .collect()
    });
    if e.is_empty(){
        return Err("No events found".to_string());
    }else{
        return Ok(e);
    }
}

pub fn count_all_events_db() -> Result<u64,String> {
    Ok(EVENTS.with(|events| events.borrow().len() as u64))
}


pub fn save_event_db(event: Event) -> Result<(),String> {
    EVENTS.with(|events| {
        events.borrow_mut().insert(event.id.clone(), event);
    });
    Ok(())
}


pub fn save_events_db(events: Vec<Event>) -> Result<(),String> {
    EVENTS.with(|events_map| {
        for event in events {
            events_map.borrow_mut().insert(event.id.clone(), event);
        }
    });
    Ok(())
}


pub fn delete_event_db(id: String) -> Result<(),String> {
    EVENTS.with(|events| {
        events.borrow_mut().remove(&id);
    });
    Ok(())
}


pub fn get_events_db(filter: Filter) -> Result<Vec<Event>,String> {
    let result : Vec<Event>= EVENTS.with(|events| {
        events
            .borrow()
            .iter()
            .filter(|(_, event)| event.is_match(&filter))
            .map(|(_, event)| event.clone())
            .collect()
    });
    if result.is_empty(){
        return Err("No events found".to_string());
    }else{
        return Ok(result);
    }
}


pub fn count_events_db(filter: Filter) -> Result<u64,String> {
    match get_events_db(filter) {
        Ok(events) => Ok(events.len() as u64),
        Err(e) => Err(e),
    }
}


pub fn clear_events_db() -> Result<(),String> {
    EVENTS.with(|events| {
        // Replace the contents of `events` with a new, empty `StableBTreeMap`.
        *events.borrow_mut() = StableBTreeMap::init(MEMORY_MANAGER.with(|p| p.borrow().get(MemoryId::new(0))));
    });
    Ok(())
}
