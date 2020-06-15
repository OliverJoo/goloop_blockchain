package org.aion.avm.core;

import foundation.icon.ee.types.Address;
import foundation.icon.ee.types.Result;
import foundation.icon.ee.types.StepCost;

import java.io.IOException;
import java.math.BigInteger;
import java.util.function.IntConsumer;

/**
 * An interface into some external component that maintains and can answer state queries pertaining
 * to the blockchain.
 */
public interface IExternalState {
    int OPTION_READ_ONLY = 1;
    int OPTION_TRACE = 2;

    /**
     * Returns the pre-transformed code associated with current score.
     *
     * Returns {@code null} if the context has no pre-transformed code.
     *
     * @return the pre-transformed code or null.
     */
    byte[] getCode() throws IOException;

    /**
     * Returns the transformed code associated with current score.
     *
     * Returns {@code null} if the context has no transformed code.
     *
     * @return the transformed code or null.
     */
    byte[] getTransformedCode() throws IOException;

    /**
     * Saves the specified transformed code associated with current score.
     *
     * @param code The code corresponding to the address.
     */
    void setTransformedCode(byte[] code);

    /**
     * Saves the specified serialized bytes of the object graph to current score.
     *
     * @param objectGraph The bytes of the object graph.
     */
    void putObjectGraph(byte[] objectGraph);

    /**
     * Returns the serialized bytes of the object graph associated with current score.
     *
     * Returns {@code null} if the address has no object graph.
     *
     * @return the serialized bytes of the object graph or null.
     */
    byte[] getObjectGraph();

    /**
     * Saves the specified key-value pairing to the given address.
     *
     * If the specified key already exists as a key-value pairing for the given address, then that
     * pairing will be updated so that its old corresponding value is replaced by the new one.
     *
     * @param key The key.
     * @param value The value.
     * @param prevSizeCB Previous size callback. Negative value is passed if
     *                   there is no previous value.
     */
    void putStorage(byte[] key, byte[] value, IntConsumer prevSizeCB);

    /**
     * Waits for a pending callback.
     *
     * Immediately returns false if there is no callback to wait.
     *
     * @return false if there is no callback to wait.
     */
    boolean waitForCallback();

    /**
     * Waits for all pending callbacks.
     */
    void waitForCallbacks();

    /**
     * Returns the value in the key-value pairing to the specified key for the given address if any
     * such pairing exists.
     *
     * Returns {@code null} otherwise, if no such key corresponds to the address.
     *
     * @param key The key.
     * @return the value or null if there is no such value.
     */
    byte[] getStorage(byte[] key);

    /**
     * Returns the balance of the specified address.
     *
     * Returns {@link BigInteger#ZERO} if the specified address has no state associated with it.
     *
     * @param address The address whose balance is to be queried.
     * @return the account balance.
     */
    BigInteger getBalance(Address address);

    /**
     * Returns the block height of the current block.
     *
     * @return the current block height.
     */
    long getBlockHeight();

    /**
     * Returns the timestamp of the current block.
     *
     * @return the current block timestamp.
     */
    long getBlockTimestamp();

    /**
     * Returns the address of the contract owner
     *
     * @return the owner address
     */
    Address getOwner();

    /**
     * Emits events log
     */
    void log(byte[][] indexed, byte[][]data);

    /**
     * Calls external method of target contract.
     */
    Result call(Address address,
                       String method,
                       Object[] params,
                       BigInteger value,
                       int stepLimit);

    int getOption();

    default boolean isReadOnly() {
        return (getOption() & OPTION_READ_ONLY) != 0;
    }

    default boolean isTrace() {
        return (getOption() & OPTION_TRACE) != 0;
    }

    StepCost getStepCost();
}
