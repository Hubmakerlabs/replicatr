use std::{cell::RefCell, collections::HashMap};

use structs::Event;

pub mod methods;
pub mod structs;

thread_local! {
    // pub static EVENTS: RefCell<Vec<Event>> = RefCell::new(Vec::new());
    pub static EVENTS: RefCell<HashMap<String, Event>> = RefCell::new(HashMap::new());
}
