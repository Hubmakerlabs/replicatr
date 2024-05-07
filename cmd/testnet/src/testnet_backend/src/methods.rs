use crate::{
    structs::{Event, Filter},
    EVENTS,
    MEMORY_MANAGER,
    db,
    acl
};
use candid::export_service;
use ic_cdk_macros::{query, update};
use ic_stable_structures::StableBTreeMap;
use ic_stable_structures::memory_manager::MemoryId;
use ic_cdk::api;
use crate::acl::{is_user, is_owner};





#[query(guard = "is_user")]
fn test() -> String {
    "Hello, world!".to_string()
}

#[query(guard = "is_user")]
fn get_all_events() -> Vec<(String, Event)> {
    db::get_all_events_db()
}

#[query(guard = "is_user")]
fn count_all_events() -> u64 {
    db::count_all_events_db()
}

#[update(guard = "is_user")]
fn save_event(event: Event) -> String {
    db::save_event_db(event)
}

#[update(guard = "is_user")]
fn save_events(events: Vec<Event>) -> String {
    db::save_events_db(events)
}

#[update(guard = "is_user")]
fn delete_event(id: String) -> String {
    db::delete_event_db(id)
}

#[query(guard = "is_user")]
fn get_events(filter: Filter) -> Vec<Event> {
    db::get_events_db(filter)
}

#[query(guard = "is_user")]
fn count_events(filter: Filter) -> u64 {
    db::count_events_db(filter)
}

#[query(name = "__get_candid_interface_tmp_hack")]
pub fn export_did_tmp_() -> String {
    export_service!();
    __export_service()
}

#[update(guard = "is_user")]
fn clear_events() -> String {
    db::clear_events_db()
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
        export_did_tmp_(),
    )
    .expect("Write failed.");
}

#[update(guard = "is_owner")]
pub fn add_user(pub_key : String,perm : bool) -> String {
    acl::add_user_acl(pub_key, perm)
}     

#[update(guard = "is_owner")]
pub fn remove_user(pub_key : String) -> String{
    acl::remove_user_acl(pub_key)
}


#[query]
pub fn get_permission() -> String {
    acl::get_permission_acl()
}

fn check_timestamp(timestamp: i64) -> bool {
    let current_time = api::time() as i64; // Convert current_time to i64
    let diff = (timestamp - current_time).abs();
    if diff > 30 {
        return false;
    }
    true
}
