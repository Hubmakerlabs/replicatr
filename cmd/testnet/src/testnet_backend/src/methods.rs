

use crate::{
    structs::{Event, Filter},
    db,
    acl
};
use candid::export_service;
use ic_cdk_macros::{query, update};
use ic_cdk::api;
use crate::acl::{is_user, is_owner};







#[query(guard = "is_user")]
pub fn get_all_events(timestamp: i64) -> Vec<(String, Event)> {
    if !valid_timestamp(timestamp){
        ic_cdk::trap(&format!("{} is an invalid timestamp", timestamp));
    }
    let result: Vec<(String, Event)>;
    match db::get_all_events_db(){
        Ok(events) => result = events,
        Err(e) => ic_cdk::trap(&format!("Error: {}", e))
    };
    result
}

#[query(guard = "is_user")]
pub fn count_all_events(timestamp: i64) -> u64 {
    if !valid_timestamp(timestamp){
        ic_cdk::trap(&format!("{} is an invalid timestamp", timestamp));
    }
    let result: u64;
    match db::count_all_events_db(){
        Ok(count) => result = count,
        Err(e) => ic_cdk::trap(&format!("Error: {}", e))
    };
    result  
}

#[update(guard = "is_user")]
pub fn save_event(event: Event,timestamp: i64) -> Option<String> {
    if !valid_timestamp(timestamp){
        let e = format!("{} is an invalid timestamp", timestamp);
        ic_cdk::trap(&e);
    }
    if let Err(e) = db::save_event_db(event){
        ic_cdk::trap(&format!("Error: {}", e));
    }
    None
}

#[update(guard = "is_user")]
pub fn save_events(events: Vec<Event>,timestamp: i64) -> Option<String> {
    if !valid_timestamp(timestamp){
        ic_cdk::trap(&format!("{} is an invalid timestamp", timestamp));
    }
    if let Err(e) = db::save_events_db(events){
        ic_cdk::trap(&format!("Error: {}", e));
    };
    None
}

#[update(guard = "is_user")]
pub fn delete_event(id: String,timestamp: i64) -> Option<String> {
    if !valid_timestamp(timestamp){
        ic_cdk::trap(&format!("{} is an invalid timestamp", timestamp));
    }
    if let Err(e) = db::delete_event_db(id){
        ic_cdk::trap(&format!("Error: {}", e));
    };
    None
}

#[query(guard = "is_user")]
pub fn get_events(filter: Filter,timestamp: i64) -> Vec<Event> {
    if !valid_timestamp(timestamp){
        ic_cdk::trap(&format!("{} is an invalid timestamp", timestamp));
    }
    let result: Vec<Event>;
    match db::get_events_db(filter){
        Ok(events) => result = events,
        Err(e) => ic_cdk::trap(&format!("Error: {}", e))
    };
    result
}

#[query(guard = "is_user")]
pub fn count_events(filter: Filter,timestamp: i64) -> u64 {
    if !valid_timestamp(timestamp){
        ic_cdk::trap(&format!("{} is an invalid timestamp", timestamp));
    }
    let result: u64;
    match db::count_events_db(filter){
        Ok(count) => result = count,
        Err(e) => ic_cdk::trap(&format!("Error: {}", e))
    };
    result
}

#[query(name = "__get_candid_interface_tmp_hack")]
pub fn export_did_tmp_() -> String {
    export_service!();
    __export_service()
}

#[update(guard = "is_user")]
pub fn clear_events(timestamp: i64) -> Option<String> {
    if !valid_timestamp(timestamp){
        ic_cdk::trap(&format!("{} is an invalid timestamp", timestamp));
    }
    if let Err(e) = db::clear_events_db(){
        ic_cdk::trap(&format!("Error: {}", e));
    };
    None
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
pub fn add_user(pub_key : String,perm : bool,timestamp: i64) -> Option<String> {
    if !valid_timestamp(timestamp){
        ic_cdk::trap(&format!("{} is an invalid timestamp", timestamp));
    }
    if let Err(e) = acl::add_user_acl(pub_key, perm){
        ic_cdk::trap(&format!("Error: {}", e));
    };
    None
}     

#[update(guard = "is_owner")]
pub fn remove_user(pub_key : String,timestamp: i64) -> Option<String>{
    if !valid_timestamp(timestamp){
        ic_cdk::trap(&format!("{} is an invalid timestamp", timestamp));
    }
    if let Err(e) = acl::remove_user_acl(pub_key){
        ic_cdk::trap(&format!("Error: {}", e));
    };
    None
}


#[query]
pub fn get_permission(timestamp: i64) -> String {
    if !valid_timestamp(timestamp){
        ic_cdk::trap(&format!("{} is an invalid timestamp", timestamp));
    }
    let result: String;
    match acl::get_permission_acl(){
        Ok(permission) => result = permission,
        Err(e) => ic_cdk::trap(&format!("Permission: {}", e))
    };
    result
}

fn valid_timestamp(timestamp: i64) -> bool {
    let current_time = api::time() as i64; // Convert current_time to i64
    let diff = (timestamp - current_time).abs();
    if diff > 30_000_000_000 {
        return false;
    }
    true
}
